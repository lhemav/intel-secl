/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package controllers_test

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/intel-secl/intel-secl/v3/pkg/hvs/controllers"
	mocks2 "github.com/intel-secl/intel-secl/v3/pkg/hvs/domain/mocks"
	hvsRoutes "github.com/intel-secl/intel-secl/v3/pkg/hvs/router"
	"github.com/intel-secl/intel-secl/v3/pkg/model/hvs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"net/http/httptest"
)

var _ = Describe("FlavorController", func() {
	var router *mux.Router
	var w *httptest.ResponseRecorder
	var flavorStore *mocks2.MockFlavorStore
	var flavorController *controllers.FlavorController
	BeforeEach(func() {
		router = mux.NewRouter()
		flavorStore = mocks2.NewFakeFlavorStore()
		flavorController = &controllers.FlavorController{Store: flavorStore}
	})

	// Specs for HTTP Get to "/flavors"
	Describe("Search Flavors", func() {
		Context("When no filter arguments are passed", func() {
			It("All Flavors records are returned", func() {
				router.Handle("/flavors", hvsRoutes.ErrorHandler(hvsRoutes.ResponseHandler(flavorController.Search))).Methods("GET")
				req, err := http.NewRequest("GET", "/flavors", nil)
				Expect(err).NotTo(HaveOccurred())
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(http.StatusOK))

				var sfs []*hvs.SignedFlavor
				err = json.Unmarshal(w.Body.Bytes(), &sfs)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(sfs)).To(Equal(1))
			})
		})

		Context("When filtered by Flavor id", func() {
			It("Should get a single flavor entry", func() {
				router.Handle("/flavors", hvsRoutes.ErrorHandler(hvsRoutes.ResponseHandler(flavorController.Search))).Methods("GET")
				req, err := http.NewRequest("GET", "/flavors?id=c36b5412-8c02-4e08-8a74-8bfa40425cf3", nil)
				Expect(err).NotTo(HaveOccurred())
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(http.StatusOK))

				var sfs []*hvs.SignedFlavor
				err = json.Unmarshal(w.Body.Bytes(), &sfs)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(sfs)).To(Equal(1))
			})
		})
		Context("When filtered by Flavor meta description key-value pair", func() {
			It("Should get a single flavor entry", func() {
				router.Handle("/flavors", hvsRoutes.ErrorHandler(hvsRoutes.ResponseHandler(flavorController.Search))).Methods("GET")
				req, err := http.NewRequest("GET", "/flavors?key=bios_name&&value=Intel Corporation", nil)
				Expect(err).NotTo(HaveOccurred())
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(http.StatusOK))

				var sfs []*hvs.SignedFlavor
				err = json.Unmarshal(w.Body.Bytes(), &sfs)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(sfs)).To(Equal(1))
			})
		})
	})

	// Specs for HTTP Get to "/flavors/{flavor_id}"
	Describe("Retrieve Flavor", func() {
		Context("Retrieve Flavor by valid ID from data store", func() {
			It("Should retrieve Flavor", func() {
				router.Handle("/flavors/{id}", hvsRoutes.ErrorHandler(hvsRoutes.ResponseHandler(flavorController.Retrieve))).Methods("GET")
				req, err := http.NewRequest("GET", "/flavors/c36b5412-8c02-4e08-8a74-8bfa40425cf3", nil)
				Expect(err).NotTo(HaveOccurred())
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(http.StatusOK))
			})
		})
		Context("Try to retrieve Flavor by non-existent ID from data store", func() {
			It("Should fail to retrieve Flavor", func() {
				router.Handle("/flavors/{id}", hvsRoutes.ErrorHandler(hvsRoutes.ResponseHandler(flavorController.Retrieve))).Methods("GET")
				req, err := http.NewRequest("GET", "/flavors/73755fda-c910-46be-821f-e8ddeab189e9", nil)
				Expect(err).NotTo(HaveOccurred())
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(http.StatusNotFound))

				var sfs []*hvs.SignedFlavor
				err = json.Unmarshal(w.Body.Bytes(), &sfs)
				Expect(err).To(HaveOccurred())
				Expect(sfs).To(BeNil())
			})
		})
	})

	// Specs for HTTP Delete to "/flavors/{flavorId}"
	Describe("Delete Flavor by ID", func() {
		Context("Delete Flavor by ID from data store", func() {
			It("Should delete Flavor", func() {
				router.Handle("/flavors/{id}", hvsRoutes.ErrorHandler(hvsRoutes.ResponseHandler(flavorController.Delete))).Methods("DELETE")
				req, err := http.NewRequest("DELETE", "/flavors/c36b5412-8c02-4e08-8a74-8bfa40425cf3", nil)
				Expect(err).NotTo(HaveOccurred())
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(204))
			})
		})
		Context("Delete Flavor by incorrect ID from data store", func() {
			It("Should fail to delete Flavor", func() {
				router.Handle("/flavors/{id}", hvsRoutes.ErrorHandler(hvsRoutes.ResponseHandler(flavorController.Delete))).Methods("DELETE")
				req, err := http.NewRequest("DELETE", "/flavors/73755fda-c910-46be-821f", nil)
				Expect(err).NotTo(HaveOccurred())
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(404))
			})
		})
		Context("Delete Flavor by invalid ID from data store", func() {
			It("Should fail to delete Flavor", func() {
				router.Handle("/flavors/{id}", hvsRoutes.ErrorHandler(hvsRoutes.ResponseHandler(flavorController.Delete))).Methods("DELETE")
				req, err := http.NewRequest("DELETE", "/flavors/73755fda-c910-46be-821f-e8ddeab189e9", nil)
				Expect(err).NotTo(HaveOccurred())
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(404))
			})
		})
	})
})