/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package interpreter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestCompositeDeferredDecoding(t *testing.T) {

	t.Parallel()

	t.Run("Simple composite", func(t *testing.T) {

		stringValue := NewStringValue("hello")
		stringValue.modified = false

		members := NewStringValueOrderedMap()
		members.Set("a", stringValue)
		members.Set("b", BoolValue(true))

		value := NewCompositeValue(
			utils.TestLocation,
			"TestResource",
			common.CompositeKindResource,
			members,
			nil,
		)

		encoded, _, err := EncodeValue(value, nil, true, nil)
		require.NoError(t, err)

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &CompositeValue{}, decoded)
		compositeValue := decoded.(*CompositeValue)

		// Value must not be loaded. i.e: the content is available
		assert.NotNil(t, compositeValue.content)

		// The meta-info and fields raw content are not loaded yet
		assert.Nil(t, compositeValue.fieldsContent)
		assert.Empty(t, compositeValue.location)
		assert.Empty(t, compositeValue.qualifiedIdentifier)
		assert.Equal(t, common.CompositeKindUnknown, compositeValue.kind)

		// Use the Getters and see whether the meta-info are loaded
		assert.Equal(t, value.Location(), compositeValue.Location())
		assert.Equal(t, value.QualifiedIdentifier(), compositeValue.QualifiedIdentifier())
		assert.Equal(t, value.Kind(), compositeValue.Kind())

		// Now the content must be cleared
		assert.Nil(t, compositeValue.content)

		// And the fields raw content must be available
		assert.NotNil(t, compositeValue.fieldsContent)

		// Check all the fields using getters

		decodedFields := compositeValue.Fields()
		require.Equal(t, 2, decodedFields.Len())

		decodeFieldValue, contains := decodedFields.Get("a")
		assert.True(t, contains)
		assert.Equal(t, stringValue, decodeFieldValue)

		decodeFieldValue, contains = decodedFields.Get("b")
		assert.True(t, contains)
		assert.Equal(t, BoolValue(true), decodeFieldValue)

		// Once all the fields are loaded, the fields raw content must be cleared
		assert.Nil(t, compositeValue.fieldsContent)
	})

	t.Run("Nested composite", func(t *testing.T) {
		value := newTestLargeCompositeValue(0)

		encoded, _, err := EncodeValue(value, nil, true, nil)
		require.NoError(t, err)

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &CompositeValue{}, decoded)
		compositeValue := decoded.(*CompositeValue)

		address, ok := compositeValue.Fields().Get("address")
		assert.True(t, ok)

		require.IsType(t, &CompositeValue{}, address)
		nestedCompositeValue := address.(*CompositeValue)

		// Inner composite value must not be loaded
		assert.NotNil(t, nestedCompositeValue.content)
	})

	t.Run("Field update", func(t *testing.T) {
		value := newTestLargeCompositeValue(0)

		encoded, _, err := EncodeValue(value, nil, true, nil)
		require.NoError(t, err)

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &CompositeValue{}, decoded)
		compositeValue := decoded.(*CompositeValue)

		newValue := NewStringValue("green")
		compositeValue.SetMember(nil, nil, "status", newValue)

		// Composite value must be loaded
		assert.Nil(t, compositeValue.content)

		// check updated value
		fieldValue, contains := compositeValue.Fields().Get("status")
		assert.True(t, contains)
		assert.Equal(t, newValue, fieldValue)
	})

	t.Run("Round trip - without loading", func(t *testing.T) {

		stringValue := NewStringValue("hello")
		stringValue.modified = false

		members := NewStringValueOrderedMap()
		members.Set("a", stringValue)
		members.Set("b", BoolValue(true))

		value := NewCompositeValue(
			utils.TestLocation,
			"TestResource",
			common.CompositeKindResource,
			members,
			nil,
		)

		// Encode
		encoded, _, err := EncodeValue(value, nil, true, nil)
		require.NoError(t, err)

		// Decode
		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		// Value must not be loaded. i.e: the content is available
		require.IsType(t, &CompositeValue{}, decoded)
		compositeValue := decoded.(*CompositeValue)
		assert.NotNil(t, compositeValue.content)

		// Re encode the decoded value
		reEncoded, _, err := EncodeValue(decoded, nil, true, nil)
		require.NoError(t, err)

		reDecoded, err := DecodeValue(reEncoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &CompositeValue{}, reDecoded)
		compositeValue = reDecoded.(*CompositeValue)

		compositeValue.ensureFieldsLoaded()

		// Check the meta info
		assert.Equal(t, value.Location(), compositeValue.Location())
		assert.Equal(t, value.QualifiedIdentifier(), compositeValue.QualifiedIdentifier())
		assert.Equal(t, value.Kind(), compositeValue.Kind())

		// Check the fields

		decodedFields := compositeValue.Fields()
		require.Equal(t, 2, decodedFields.Len())

		decodeFieldValue, contains := decodedFields.Get("a")
		assert.True(t, contains)
		assert.Equal(t, stringValue, decodeFieldValue)

		decodeFieldValue, contains = decodedFields.Get("b")
		assert.True(t, contains)
		assert.Equal(t, BoolValue(true), decodeFieldValue)
	})

	t.Run("Round trip - partially loaded", func(t *testing.T) {

		stringValue := NewStringValue("hello")
		stringValue.modified = false

		members := NewStringValueOrderedMap()
		members.Set("a", stringValue)
		members.Set("b", BoolValue(true))

		value := NewCompositeValue(
			utils.TestLocation,
			"TestResource",
			common.CompositeKindResource,
			members,
			nil,
		)

		// Encode
		encoded, _, err := EncodeValue(value, nil, true, nil)
		require.NoError(t, err)

		// Decode
		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		// Partially loaded the value.

		require.IsType(t, &CompositeValue{}, decoded)
		compositeValue := decoded.(*CompositeValue)
		// This will only load the meta info, but not the fields
		compositeValue.QualifiedIdentifier()

		assert.Nil(t, compositeValue.content)
		assert.NotNil(t, compositeValue.fieldsContent)

		// Re encode the decoded value
		reEncoded, _, err := EncodeValue(decoded, nil, true, nil)
		require.NoError(t, err)

		// Decode back the value
		reDecoded, err := DecodeValue(reEncoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &CompositeValue{}, reDecoded)
		compositeValue = reDecoded.(*CompositeValue)

		compositeValue.ensureFieldsLoaded()

		// Check the meta info
		assert.Equal(t, value.Location(), compositeValue.Location())
		assert.Equal(t, value.QualifiedIdentifier(), compositeValue.QualifiedIdentifier())
		assert.Equal(t, value.Kind(), compositeValue.Kind())

		// Check the fields

		decodedFields := compositeValue.Fields()
		require.Equal(t, 2, decodedFields.Len())

		decodeFieldValue, contains := decodedFields.Get("a")
		assert.True(t, contains)
		assert.Equal(t, stringValue, decodeFieldValue)

		decodeFieldValue, contains = decodedFields.Get("b")
		assert.True(t, contains)
		assert.Equal(t, BoolValue(true), decodeFieldValue)
	})
}

func BenchmarkCompositeDeferredDecoding(b *testing.B) {

	encoded, _, err := EncodeValue(newTestLargeCompositeValue(0), nil, true, nil)
	require.NoError(b, err)

	b.Run("Simply decode", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
			require.NoError(b, err)
		}
	})

	b.Run("Access identifier", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
			require.NoError(b, err)

			composite := decoded.(*CompositeValue)
			composite.QualifiedIdentifier()
		}
	})

	b.Run("Access field", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
			require.NoError(b, err)

			composite := decoded.(*CompositeValue)
			_, ok := composite.Fields().Get("fname")
			require.True(b, ok)
		}
	})

	b.Run("Re-encode decoded", func(b *testing.B) {
		b.ReportAllocs()

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(b, err)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _, err = EncodeValue(decoded, nil, true, nil)
			require.NoError(b, err)
		}
	})
}

var newTestLargeCompositeValue = func(id int) *CompositeValue {
	addressFields := NewStringValueOrderedMap()
	addressFields.Set("street", NewStringValue(fmt.Sprintf("No: %d", id)))
	addressFields.Set("city", NewStringValue("Vancouver"))
	addressFields.Set("state", NewStringValue("BC"))
	addressFields.Set("country", NewStringValue("Canada"))

	address := NewCompositeValue(
		utils.TestLocation,
		"Address",
		common.CompositeKindStructure,
		addressFields,
		nil,
	)

	members := NewStringValueOrderedMap()
	members.Set("fname", NewStringValue("John"))
	members.Set("lname", NewStringValue("Doe"))
	members.Set("age", NewIntValueFromInt64(999))
	members.Set("status", NewStringValue("unknown"))
	members.Set("address", address)

	return NewCompositeValue(
		utils.TestLocation,
		"Person",
		common.CompositeKindStructure,
		members,
		nil,
	)
}
