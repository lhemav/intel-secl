/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */

package hrrs

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/intel-secl/intel-secl/v3/pkg/hvs/domain"
	"github.com/intel-secl/intel-secl/v3/pkg/hvs/domain/mocks"
	"github.com/intel-secl/intel-secl/v3/pkg/hvs/domain/models"
	"github.com/intel-secl/intel-secl/v3/pkg/model/hvs"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	tenSeconds, _      = time.ParseDuration("10s")
	twoSeconds, _      = time.ParseDuration("2s")
	tenYears, _        = time.ParseDuration("10y")
	twentyFourHours, _ = time.ParseDuration("24h")
)

func TestHostReportRefresher(t *testing.T) {

	cfg := HRRSConfig{
		RefreshPeriod: twoSeconds,
	}

	reportStore := mocks.NewEmptyMockReportStore()

	// Create a report that expired ten years ago to test the 'open window' logic.
	// Expect that this host is updated on the first pass of the report refresher.
	host1UUID := uuid.New()
	_, _ = reportStore.Create(&models.HVSReport{
		ID:         uuid.New(),
		HostID:     host1UUID,
		CreatedAt:  time.Now(),
		Expiration: time.Now().Add(-tenYears),
		TrustReport: hvs.TrustReport{
			Trusted: true,
		},
	})

	// Create another report that expires in the future to test the 'narrow windows' logic...
	// Expect that this host is updated in a secondary refresh of the report refresher.
	host2UUID := uuid.New()
	_, _ = reportStore.Create(&models.HVSReport{
		ID:         uuid.New(),
		HostID:     host2UUID,
		CreatedAt:  time.Now(),
		Expiration: time.Now().Add(twoSeconds * 3),
		TrustReport: hvs.TrustReport{
			Trusted: true,
		},
	})

	hostTrustManager := MockHostTrustManager{
		reportStore: reportStore,
	}

	// create a new HostReportRefresher, 'run' the backgound thread and then
	// sleep for ten seconds.  We expect the expired report to be updated
	// in the report store.
	refresher, err := NewHostReportRefresher(cfg, reportStore, hostTrustManager)
	assert.NoError(t, err)
	refresher.Run()

	time.Sleep(tenSeconds)

	t.Log("stopping")
	refresher.Stop()

	// make sure both hosts have updated reports with future expiration dates
	hostsToCheck := []uuid.UUID{host1UUID, host2UUID}
	for _, hostId := range hostsToCheck {
		criteria := models.ReportFilterCriteria{
			HostID: hostId,
		}

		reports, err := reportStore.Search(&criteria)
		assert.NoError(t, err)

		assert.NotNil(t, reports)
		assert.Equal(t, len(reports), 1)
		assert.True(t, reports[0].Expiration.After(time.Now()))
	}
}

//-------------------------------------------------------------------------------------------------
// M O C K   H O S T   T R U S T   M A N A G E R
//-------------------------------------------------------------------------------------------------
type MockHostTrustManager struct {
	reportStore domain.ReportStore
}

func (htm MockHostTrustManager) VerifyHost(hostId uuid.UUID, fetchHostData, preferHashMatch bool) (*models.HVSReport, error) {
	return nil, errors.New("VerifyHost is not implemented")
}

func (htm MockHostTrustManager) ProcessQueue() error {
	return errors.New("ProcessQueue is not implemented")
}

func (htm MockHostTrustManager) VerifyHostsAsync(hostIDs []uuid.UUID, fetchHostData, preferHashMatch bool) error {

	for _, hostID := range hostIDs {

		// simulate removing the old report and create a new one that
		// expires in 24 hours
		criteria := models.ReportFilterCriteria{
			HostID: hostID,
		}

		reportsToDelete, err := htm.reportStore.Search(&criteria)
		if err != nil {
			return errors.Wrap(err, "There was an error searching for the report by host id")
		}

		for _, reportToDelete := range reportsToDelete {
			htm.reportStore.Delete(reportToDelete.ID)
		}

		trustReport := models.HVSReport{
			ID:         uuid.New(),
			HostID:     hostID,
			CreatedAt:  time.Now(),
			Expiration: time.Now().Add(twentyFourHours),
			TrustReport: hvs.TrustReport{
				Trusted: true,
			},
		}

		_, err = htm.reportStore.Create(&trustReport)
		if err != nil {
			return nil
		}
	}

	return nil
}
