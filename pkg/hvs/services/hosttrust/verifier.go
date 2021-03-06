/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */

package hosttrust

import (
	"github.com/google/uuid"
	"github.com/intel-secl/intel-secl/v3/pkg/hvs/domain"
	"github.com/intel-secl/intel-secl/v3/pkg/hvs/domain/models"
	"github.com/intel-secl/intel-secl/v3/pkg/lib/host-connector/types"
	"github.com/intel-secl/intel-secl/v3/pkg/lib/saml"
	flavorVerifier "github.com/intel-secl/intel-secl/v3/pkg/lib/verifier"
	"github.com/intel-secl/intel-secl/v3/pkg/model/hvs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var ErrInvalidHostManiFest = errors.New("invalid host data")
var ErrManifestMissingHwUUID = errors.New("host data missing hardware uuid")

type Verifier struct {
	FlavorStore                     domain.FlavorStore
	FlavorGroupStore                domain.FlavorGroupStore
	HostStore                       domain.HostStore
	ReportStore                     domain.ReportStore
	FlavorVerifier                  flavorVerifier.Verifier
	CertsStore                      models.CertificatesStore
	SamlIssuer                      saml.IssuerConfiguration
	SkipFlavorSignatureVerification bool
}

func NewVerifier(cfg domain.HostTrustVerifierConfig) domain.HostTrustVerifier {
	return &Verifier{
		FlavorStore:                     cfg.FlavorStore,
		FlavorGroupStore:                cfg.FlavorGroupStore,
		HostStore:                       cfg.HostStore,
		ReportStore:                     cfg.ReportStore,
		FlavorVerifier:                  cfg.FlavorVerifier,
		CertsStore:                      cfg.CertsStore,
		SamlIssuer:                      cfg.SamlIssuerConfig,
		SkipFlavorSignatureVerification: cfg.SkipFlavorSignatureVerification,
	}
}

func (v *Verifier) Verify(hostId uuid.UUID, hostData *types.HostManifest, newData bool) (*models.HVSReport, error) {
	defaultLog.Trace("hosttrust/verifier:Verify() Entering")
	defer defaultLog.Trace("hosttrust/verifier:Verify() Leaving")
	if hostData == nil {
		return nil, ErrInvalidHostManiFest
	}
	//TODO: Fix HardwareUUID has to be uuid
	hwUuid, err := uuid.Parse(hostData.HostInfo.HardwareUUID)
	if err != nil || hwUuid == uuid.Nil {
		return nil, ErrManifestMissingHwUUID
	}

	// TODO : remove this when we remove the intermediate collection
	flvGroupIds, err := v.HostStore.SearchFlavorgroups(hostId)
	flvGroups, err := v.FlavorGroupStore.Search(&models.FlavorGroupFilterCriteria{Ids: flvGroupIds})
	if err != nil {
		return nil, errors.New("hosttrust/verifier:Verify() Store access error")
	}
	// start with the presumption that final trust report would be true. It as some point, we get an invalid report,
	// the Overall trust status would be negative
	var finalReportValid = true // This is the final trust report - initialize
	// create an empty trust report with the host manifest
	finalTrustReport := hvs.TrustReport{HostManifest: *hostData}

	for _, fg := range flvGroups {
		//TODO - handle errors in case of DB transaction
		fgTrustReqs, err := NewFlvGrpHostTrustReqs(hostId, hwUuid, fg, v.FlavorStore, hostData, v.SkipFlavorSignatureVerification)
		if err != nil {
			return nil, errors.Wrap(err, "hosttrust/verifier:Verify() Error while retrieving NewFlvGrpHostTrustReqs")
		}
		fgCachedFlavors, err := v.getCachedFlavors(hostId, (fg).ID)
		if err != nil {
			return nil, errors.Wrap(err, "hosttrust/verifier:Verify() Error while retrieving getCachedFlavors")
		}

		var fgTrustCache hostTrustCache
		if len(fgCachedFlavors) > 0 {
			fgTrustCache, err = v.validateCachedFlavors(hostId, hostData, fgCachedFlavors)
			if err != nil {
				return nil, errors.Wrap(err, "hosttrust/verifier:Verify() Error while validating cache")
			}
		}

		fgTrustReport := fgTrustCache.trustReport
		if !fgTrustReqs.MeetsFlavorGroupReqs(fgTrustCache, v.FlavorVerifier.GetVerifierCerts()) {
			log.Debug("hosttrust/verifier:Verify() Trust cache doesn't meet flavorgroup requirements")
			finalReportValid = false
			fgTrustReport, err = v.CreateFlavorGroupReport(hostId, *fgTrustReqs, hostData, fgTrustCache)
			if err != nil {
				return nil, errors.Wrap(err, "hosttrust/verifier:Verify() Error while creating flavorgroup report")
			}
		}
		log.Debug("hosttrust/verifier:Verify() Trust status for host id ", hostId, " for flavorgroup ", fg.ID, " is ", fgTrustReport.IsTrusted())
		// append the results
		finalTrustReport.AddResults(fgTrustReport.Results)
	}
	// create a new report if we actually have any results and either the Final Report is untrusted or
	// we have new Data from the host and therefore need to update based on the new report.
	var hvsReport *models.HVSReport
	log.Debugf("hosttrust/verifier:Verify() Final results in report: %d", len(finalTrustReport.Results))
	if len(finalTrustReport.Results) > 0 && (!finalReportValid || newData) {
		log.Debugf("hosttrust/verifier:Verify() Generating new SAML for host: %s", hostId)
		samlReportGen := NewSamlReportGenerator(&v.SamlIssuer)
		samlReport := samlReportGen.GenerateSamlReport(&finalTrustReport)
		finalTrustReport.Trusted = finalTrustReport.IsTrusted()
		log.Debugf("hosttrust/verifier:Verify() Saving new report for host: %s", hostId)
		hvsReport = v.storeTrustReport(hostId, &finalTrustReport, &samlReport)
	}
	return hvsReport, nil
}

func (v *Verifier) getCachedFlavors(hostId uuid.UUID, flavGrpId uuid.UUID) ([]hvs.SignedFlavor, error) {
	defaultLog.Trace("hosttrust/verifier:getCachedFlavors() Entering")
	defer defaultLog.Trace("hosttrust/verifier:getCachedFlavors() Leaving")
	// retrieve the IDs of the trusted flavors from the host store
	if flIds, err := v.HostStore.RetrieveTrustCacheFlavors(hostId, flavGrpId); err != nil && len(flIds) == 0 {
		return nil, errors.Wrap(err, "hosttrust/verifier:Verify() Error while retrieving TrustCacheFlavors")
	} else {
		result := make([]hvs.SignedFlavor, 0, len(flIds))
		for _, flvId := range flIds {
			if flv, err := v.FlavorStore.Retrieve(flvId); err == nil {
				result = append(result, *flv)
			}
		}
		return result, nil
	}
}

func (v *Verifier) validateCachedFlavors(hostId uuid.UUID,
	hostData *types.HostManifest,
	cachedFlavors []hvs.SignedFlavor) (hostTrustCache, error) {
	defaultLog.Trace("hosttrust/verifier:validateCachedFlavors() Entering")
	defer defaultLog.Trace("hosttrust/verifier:validateCachedFlavors() Leaving")

	htc := hostTrustCache{
		hostID: hostId,
	}
	var collectiveReport hvs.TrustReport
	var trustCachesToDelete []uuid.UUID
	for _, cachedFlavor := range cachedFlavors {
		//TODO: change the signature verification depending on decision on signed flavors
		report, err := v.FlavorVerifier.Verify(hostData, &cachedFlavor, v.SkipFlavorSignatureVerification)
		if err != nil {
			return hostTrustCache{}, errors.Wrap(err, "hosttrust/verifier:validateCachedFlavors() Error from flavor verifier")
		}
		if report.Trusted {
			htc.trustedFlavors = append(htc.trustedFlavors, cachedFlavor.Flavor)
			collectiveReport.Results = append(collectiveReport.Results, report.Results...)
		} else {
			trustCachesToDelete = append(trustCachesToDelete, cachedFlavor.Flavor.Meta.ID)
		}
	}
	// remove cache entries for flavors that could not be verified
	_ = v.HostStore.RemoveTrustCacheFlavors(hostId, trustCachesToDelete)
	htc.trustReport = collectiveReport
	return htc, nil
}

func (v *Verifier) storeTrustReport(hostID uuid.UUID, trustReport *hvs.TrustReport, samlReport *saml.SamlAssertion) *models.HVSReport {
	defaultLog.Trace("hosttrust/verifier:storeTrustReport() Entering")
	defer defaultLog.Trace("hosttrust/verifier:storeTrustReport() Leaving")

	log.Debugf("hosttrust/verifier:storeTrustReport() flavorverify host: %s SAML Report: %s", hostID, samlReport.Assertion)
	hvsReport := models.HVSReport{
		HostID:      hostID,
		TrustReport: *trustReport,
		CreatedAt:   samlReport.CreatedTime,
		Expiration:  samlReport.ExpiryTime,
		Saml:        samlReport.Assertion,
	}
	report, err := v.ReportStore.Update(&hvsReport)
	if err != nil {
		log.WithError(err).Errorf("hosttrust/verifier:storeTrustReport() Failed to store Report")
	}
	return report
}
