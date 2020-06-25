/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */

package mocks

import (
	"errors"
	"github.com/google/uuid"
	"github.com/intel-secl/intel-secl/v3/pkg/hvs/domain"
	"github.com/intel-secl/intel-secl/v3/pkg/hvs/domain/models"
	"github.com/intel-secl/intel-secl/v3/pkg/model/hvs"
)

type hostStore struct {
	m map[uuid.UUID]hvs.Host
}

func NewHostStore() domain.HostStore {

	return &hostStore{make(map[uuid.UUID]hvs.Host)}
}

func (hs *hostStore) Create(host *hvs.Host) (*hvs.Host, error) {
	rec := *host
	rec.Id = uuid.New()
	copy(rec.FlavorgroupNames, host.FlavorgroupNames)
	hs.m[rec.Id] = rec
	cp := rec
	return &cp, nil
}

func (hs *hostStore) Retrieve(uuid uuid.UUID) (*hvs.Host, error) {
	if _, ok := hs.m[uuid]; ok {
		cp := hs.m[uuid]
		return &cp, nil
	}
	return nil, errors.New("Record not fouund")
}

func (hs *hostStore) Update(host *hvs.Host) (*hvs.Host, error) {
	if rec, ok := hs.m[host.Id]; ok {

		if len(host.FlavorgroupNames) > 0 {
			rec.FlavorgroupNames = append([]string{}, host.FlavorgroupNames...)
		}
		if host.ConnectionString != "" {
			rec.ConnectionString = host.ConnectionString
		}
		if host.Description != "" {
			rec.Description = host.Description
		}
		if host.HardwareUuid != uuid.Nil {
			rec.HardwareUuid = host.HardwareUuid
		}
		hs.m[host.Id] = rec
		cp := rec
		return &cp, nil
	}
	return nil, errors.New("Record not found")
}

func (hs *hostStore) Delete(uuid uuid.UUID) error {
	if _, ok := hs.m[uuid]; ok {
		delete(hs.m, uuid)
		return nil
	}
	return errors.New("Record not found")
}

func (hs *hostStore) Search(criteria *models.HostFilterCriteria) (*hvs.HostCollection, error) {
	var critId uuid.UUID
	var err error
	if criteria.Id == "" {
		critId = uuid.UUID{}
	} else {
		if critId, err = uuid.Parse(criteria.Id); err != nil {
			return nil, err
		}
	}
	if critId == uuid.Nil {
		result := make([]*hvs.Host, 0, len(hs.m))
		for _, v := range hs.m {
			result = append(result, &v)
		}
		return &hvs.HostCollection{result}, nil
	}
	if _, ok := hs.m[critId]; ok {
		cp := hs.m[critId]
		return &hvs.HostCollection{[]*hvs.Host{&cp}}, nil
	}
	return nil, errors.New("No Records fouund")
}