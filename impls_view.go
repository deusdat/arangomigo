package main

import (
	"context"
	"fmt"
	"github.com/arangodb/go-driver"
	"github.com/pkg/errors"
	"reflect"
)

// The impls_view contains all the implementation code for create, modifying and deleting an Arango
// Search View.
func (searchView SearchView) migrate(ctx context.Context, db driver.Database, extras map[string]interface{}) error {
	switch searchView.Action {
	case CREATE:
		viewProperties := buildViewProperties(searchView)
		_, err := db.CreateArangoSearchView(ctx, searchView.Name, &viewProperties)
		if !e(err) {
			fmt.Printf("Created view '%s'\n", searchView.Name)
		}
		return errors.Wrapf(err, "Couldn't create view '%s'", searchView.Name)
	case DELETE:
		view, err := db.View(ctx, searchView.Name)
		if e(err) {
			return errors.Wrapf(err, "Couldn't find view '%s' to delete", searchView.Name)
		}
		err = view.Remove(ctx)
		if !e(err) {
			fmt.Printf("Deleted view '%s'\n", searchView.Name)
		}
		return errors.Wrapf(err, "Couldn't delete view '%s'", searchView.Name)
	case MODIFY:
		view, err := db.View(ctx, searchView.Name)
		if e(err) {
			return errors.Wrapf(err, "Couldn't find view '%s' to update", searchView.Name)
		}
		aView, err := view.ArangoSearchView()
		if e(err) {
			return errors.Wrapf(err, "Couldn't get ArangoSearchView '%s' to update", searchView.Name)
		}
		viewProperties := buildViewProperties(searchView)
		err = aView.SetProperties(ctx, viewProperties)
		if !e(err) {
			fmt.Printf("Updated view '%s'\n", searchView.Name)
		}
		return errors.Wrapf(err, "Couldn't update SearchView '%s'", searchView.Name)
	}

	return nil
}

func buildViewProperties(searchView SearchView) driver.ArangoSearchViewProperties {
	viewProperties := driver.ArangoSearchViewProperties{}
	if searchView.CleanupIntervalStep != nil {
		viewProperties.CleanupIntervalStep = searchView.CleanupIntervalStep
	}
	if searchView.CommitIntervalMsec != nil {
		viewProperties.CommitInterval = searchView.CommitIntervalMsec
	}
	if searchView.ConsolidationIntervalMsec != nil {
		viewProperties.ConsolidationInterval = searchView.ConsolidationIntervalMsec
	}
	if searchView.ConsolidationPolicy != nil {
		policy := buildSearchConsolidationPolicy(searchView.ConsolidationPolicy)
		viewProperties.ConsolidationPolicy = &policy
	}
	if len(searchView.SortFields) > 0 {
		for _, field := range searchView.SortFields {
			sortField := buildSortField(field)
			viewProperties.PrimarySort = append(
				viewProperties.PrimarySort,
				sortField)
		}
	}
	if len(searchView.Links) > 0 {
		viewProperties.Links = driver.ArangoSearchLinks{}
		for _, link := range searchView.Links {
			viewProperties.Links[link.Name] = buildFields(link)
		}
	}

	return viewProperties
}

func buildSortField(field SortField) driver.ArangoSearchPrimarySortEntry {
	sortEntry := driver.ArangoSearchPrimarySortEntry{Field:field.Field}
	if field.Ascending != nil {
		direction := driver.ArangoSearchSortDirectionDesc
		if *field.Ascending {
			direction = driver.ArangoSearchSortDirectionAsc
		}
		sortEntry.Ascending = field.Ascending
		sortEntry.Direction = &direction
	}
	return sortEntry
}

func buildFields(properties SearchElementProperties) driver.ArangoSearchElementProperties {
	props := driver.ArangoSearchElementProperties{}
	if properties.IncludeAllFields != nil {
		props.IncludeAllFields = properties.IncludeAllFields
	}
	if properties.TrackListPositions != nil {
		props.TrackListPositions = properties.TrackListPositions
	}
	if properties.StoreValues != nil {
		if string(driver.ArangoSearchStoreValuesNone) == *properties.StoreValues {
			props.StoreValues = driver.ArangoSearchStoreValuesNone
		}
		if string(driver.ArangoSearchStoreValuesID) == *properties.StoreValues {
			props.StoreValues = driver.ArangoSearchStoreValuesID
		}
	}
	if len(properties.Analyzers) > 0 {
		props.Analyzers = properties.Analyzers
	}
	if len(properties.Fields) > 0 {
		fields := driver.ArangoSearchFields{}
		for _, field := range properties.Fields {
			toUse := buildFields(field)
			fields[field.Name] = toUse
		}
		props.Fields = fields
	}
	return props
}
func buildSearchConsolidationPolicy(consolidationPolicy *ConsolidationPolicy) driver.ArangoSearchConsolidationPolicy {
	policy := driver.ArangoSearchConsolidationPolicy{}
	switch consolidationPolicy.Type {
	case string(driver.ArangoSearchConsolidationPolicyTypeTier):
		policy.Type = driver.ArangoSearchConsolidationPolicyTypeTier
		if val, ok := consolidationPolicy.Options["lookahead"]; ok {
			lookahead := getInt(val)
			policy.Lookahead = &lookahead
		}
		if val, ok := consolidationPolicy.Options["maxSegments"]; ok {
			maxSegments := getInt(val)
			policy.MaxSegments = &maxSegments
		}
		if val, ok := consolidationPolicy.Options["minSegments"]; ok {
			minSegments := getInt(val)
			policy.MinSegments = &minSegments
		}
		if val, ok := consolidationPolicy.Options["segmentsBytesFloor"]; ok {
			segmentsBytesFloor := getInt(val)
			policy.SegmentsBytesFloor = &segmentsBytesFloor
		}
		if val, ok := consolidationPolicy.Options["segmentsBytesMax"]; ok {
			segmentsBytesMax := getInt(val)
			policy.SegmentsBytesMax = &segmentsBytesMax
		}
		break
	case string(driver.ArangoSearchConsolidationPolicyTypeBytesAccum):
		policy.Type = driver.ArangoSearchConsolidationPolicyTypeBytesAccum
		if val, ok := consolidationPolicy.Options["threshold"]; ok {
			threshold := getFloat(val)
			policy.Threshold = &threshold
		}
	}
	return policy
}

var floatType = reflect.TypeOf(float64(0))
var intType = reflect.TypeOf(int64(0))

func getFloat(unk interface{}) float64 {
	v := reflect.ValueOf(unk)
	v = reflect.Indirect(v)
	fv := v.Convert(floatType)
	return fv.Float()
}
func getInt(unk interface{}) int64  {
	v := reflect.ValueOf(unk)
	v = reflect.Indirect(v)
	fv := v.Convert(intType)
	return fv.Int()
}
