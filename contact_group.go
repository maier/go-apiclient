// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Contact Group API support - Fetch, Create, Update, Delete, and Search
// See: https://login.circonus.com/resources/api/calls/contact_group

package apiclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/circonus-labs/go-apiclient/config"
	"github.com/pkg/errors"
)

// ContactGroupAlertFormats define alert formats
type ContactGroupAlertFormats struct {
	LongMessage  *string `json:"long_message,omitempty"`  // string - 2019-10-09 null no longer accepted, must be omitted if not set
	LongSubject  *string `json:"long_subject,omitempty"`  // string - 2019-10-09 null no longer accepted, must be omitted if not set
	LongSummary  *string `json:"long_summary,omitempty"`  // string - 2019-10-09 null no longer accepted, must be omitted if not set
	ShortMessage *string `json:"short_message,omitempty"` // string - 2019-10-09 null no longer accepted, must be omitted if not set
	ShortSummary *string `json:"short_summary,omitempty"` // string - 2019-10-09 null no longer accepted, must be omitted if not set
}

// ContactGroupContactsExternal external contacts
type ContactGroupContactsExternal struct {
	Info   string `json:"contact_info"` // string
	Method string `json:"method"`       // string
}

// ContactGroupContactsUser user contacts
type ContactGroupContactsUser struct {
	Info    string `json:"_contact_info,omitempty"` // string
	Method  string `json:"method"`                  // string
	UserCID string `json:"user"`                    // string
}

// ContactGroupContacts list of contacts
type ContactGroupContacts struct {
	External []ContactGroupContactsExternal `json:"external"` // [] len >= 0
	Users    []ContactGroupContactsUser     `json:"users"`    // [] len >= 0
}

// ContactGroupEscalation defines escalations for severity levels
type ContactGroupEscalation struct {
	ContactGroupCID string `json:"contact_group"` // string
	After           uint   `json:"after"`         // uint
}

// ContactGroup defines a contact group. See https://login.circonus.com/resources/api/calls/contact_group for more information.
type ContactGroup struct {
	CID               string                    `json:"_cid,omitempty"`               // string
	LastModifiedBy    string                    `json:"_last_modified_by,omitempty"`  // string
	Name              string                    `json:"name,omitempty"`               // string
	GroupType         string                    `json:"group_type,omitempty"`         // string - new 2019-10-09
	AlertFormats      ContactGroupAlertFormats  `json:"alert_formats,omitempty"`      // ContactGroupAlertFormats
	Contacts          ContactGroupContacts      `json:"contacts,omitempty"`           // ContactGroupContacts
	Escalations       []*ContactGroupEscalation `json:"escalations,omitempty"`        // [] len == 5, elements: ContactGroupEscalation or null
	Reminders         []uint                    `json:"reminders,omitempty"`          // [] len == 5
	Tags              []string                  `json:"tags,omitempty"`               // [] len >= 0
	LastModified      uint                      `json:"_last_modified,omitempty"`     // uint
	AggregationWindow uint                      `json:"aggregation_window,omitempty"` // uint
	AlwaysSendClear   bool                      `json:"always_send_clear,omitempty"`  // bool - new 2019-10-09
}

// NewContactGroup returns a ContactGroup (with defaults, if applicable)
func NewContactGroup() *ContactGroup {
	return &ContactGroup{
		Escalations: make([]*ContactGroupEscalation, config.NumSeverityLevels),
		Reminders:   make([]uint, config.NumSeverityLevels),
		Contacts: ContactGroupContacts{
			External: []ContactGroupContactsExternal{},
			Users:    []ContactGroupContactsUser{},
		},
	}
}

// FetchContactGroup retrieves contact group with passed cid.
func (a *API) FetchContactGroup(cid CIDType) (*ContactGroup, error) {
	if cid == nil || *cid == "" {
		return nil, errors.New("invalid contact group CID (none)")
	}

	var groupCID string
	if !strings.HasPrefix(*cid, config.ContactGroupPrefix) {
		groupCID = fmt.Sprintf("%s/%s", config.ContactGroupPrefix, *cid)
	} else {
		groupCID = *cid
	}

	matched, err := regexp.MatchString(config.ContactGroupCIDRegex, groupCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, errors.Errorf("invalid contact group CID (%s)", groupCID)
	}

	result, err := a.Get(groupCID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching contact group")
	}

	if a.Debug {
		a.Log.Printf("fetch contact group, received JSON: %s", string(result))
	}

	group := new(ContactGroup)
	if err := json.Unmarshal(result, group); err != nil {
		return nil, errors.Wrap(err, "parsing contact group")
	}

	return group, nil
}

// FetchContactGroups retrieves all contact groups available to the API Token.
func (a *API) FetchContactGroups() (*[]ContactGroup, error) {
	result, err := a.Get(config.ContactGroupPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "fetching contact groups")
	}

	var groups []ContactGroup
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, errors.Wrap(err, "parsing contact groups")
	}

	return &groups, nil
}

// UpdateContactGroup updates passed contact group.
func (a *API) UpdateContactGroup(cfg *ContactGroup) (*ContactGroup, error) {
	if cfg == nil {
		return nil, errors.New("invalid contact group config (nil)")
	}

	groupCID := cfg.CID

	matched, err := regexp.MatchString(config.ContactGroupCIDRegex, groupCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, errors.Errorf("invalid contact group CID (%s)", groupCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("update contact group, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Put(groupCID, jsonCfg)
	if err != nil {
		return nil, errors.Wrap(err, "updating contact group")
	}

	group := &ContactGroup{}
	if err := json.Unmarshal(result, group); err != nil {
		return nil, errors.Wrap(err, "parsing contact group")
	}

	return group, nil
}

// CreateContactGroup creates a new contact group.
func (a *API) CreateContactGroup(cfg *ContactGroup) (*ContactGroup, error) {
	if cfg == nil {
		return nil, errors.New("invalid contact group config (nil)")
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("create contact group, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Post(config.ContactGroupPrefix, jsonCfg)
	if err != nil {
		return nil, errors.Wrap(err, "creating contact group")
	}

	group := &ContactGroup{}
	if err := json.Unmarshal(result, group); err != nil {
		return nil, errors.Wrap(err, "parsing contact group")
	}

	return group, nil
}

// DeleteContactGroup deletes passed contact group.
func (a *API) DeleteContactGroup(cfg *ContactGroup) (bool, error) {
	if cfg == nil {
		return false, errors.New("invalid contact group config (nil)")
	}
	return a.DeleteContactGroupByCID(CIDType(&cfg.CID))
}

// DeleteContactGroupByCID deletes contact group with passed cid.
func (a *API) DeleteContactGroupByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, errors.New("invalid contact group CID (none)")
	}

	var groupCID string
	if !strings.HasPrefix(*cid, config.ContactGroupPrefix) {
		groupCID = fmt.Sprintf("%s/%s", config.ContactGroupPrefix, *cid)
	} else {
		groupCID = *cid
	}

	matched, err := regexp.MatchString(config.ContactGroupCIDRegex, groupCID)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, errors.Errorf("invalid contact group CID (%s)", groupCID)
	}

	_, err = a.Delete(groupCID)
	if err != nil {
		return false, errors.Wrap(err, "deleting contact group")
	}

	return true, nil
}

// SearchContactGroups returns contact groups matching the specified
// search query and/or filter. If nil is passed for both parameters
// all contact groups will be returned.
func (a *API) SearchContactGroups(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]ContactGroup, error) {
	q := url.Values{}

	if searchCriteria != nil && *searchCriteria != "" {
		q.Set("search", string(*searchCriteria))
	}

	if filterCriteria != nil && len(*filterCriteria) > 0 {
		for filter, criteria := range *filterCriteria {
			for _, val := range criteria {
				q.Add(filter, val)
			}
		}
	}

	if q.Encode() == "" {
		return a.FetchContactGroups()
	}

	reqURL := url.URL{
		Path:     config.ContactGroupPrefix,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, errors.Wrap(err, "searching contact groups")
	}

	var groups []ContactGroup
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, errors.Wrap(err, "parsing contact groups")
	}

	return &groups, nil
}
