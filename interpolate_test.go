package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type InterpolateTestSuite struct {
	suite.Suite
	VariableDictionary map[string]string
}

func (suite *InterpolateTestSuite) SetupTest() {
	suite.VariableDictionary = map[string]string{
		"foo": "foo-value",
		"bar": "bar-value, ${foo}",
		"baz": "baz-value, ${bar}",
		"rec": "do${rec}"}
}

// Interpolate empty string
func (suite *InterpolateTestSuite) TestEmpty() {
	actual, err := Interpolate("", suite.VariableDictionary)
	if assert.NoError(suite.T(), err) {
		assert.Equal(suite.T(), "", actual)
	}
}

// Interpolate literal
func (suite *InterpolateTestSuite) TestLiteral() {
	actual, err := Interpolate("literal", suite.VariableDictionary)
	if assert.NoError(suite.T(), err) {
		assert.Equal(suite.T(), "literal", actual)
	}
}

func (suite *InterpolateTestSuite) TestDollerTerminated() {
	actual, err := Interpolate("foobarbaz$", suite.VariableDictionary)
	if assert.NoError(suite.T(), err) {
		assert.Equal(suite.T(), "foobarbaz$", actual)
	}
}

func (suite *InterpolateTestSuite) TestSimpleSubstitution() {
	actual, err := Interpolate("${foo}", suite.VariableDictionary)
	if assert.NoError(suite.T(), err) {
		assert.Equal(suite.T(), "foo-value", actual)
	}
}

func (suite *InterpolateTestSuite) TestNestedSubstitution() {
	actual, err := Interpolate("${baz}", suite.VariableDictionary)
	if assert.NoError(suite.T(), err) {
		assert.Equal(suite.T(), "baz-value, bar-value, foo-value", actual)
	}
}

func (suite *InterpolateTestSuite) TestDoubleDoller() {
	actual, err := Interpolate("${foo}$$${bar}", suite.VariableDictionary)
	if assert.NoError(suite.T(), err) {
		assert.Equal(suite.T(), "foo-value$bar-value, foo-value", actual)
	}
}

func (suite *InterpolateTestSuite) TestUnmatchedError() {
	_, err := Interpolate("${foo", suite.VariableDictionary)
	assert.EqualError(suite.T(), err, "Unmatched `{` found after \"{foo\".")
}

func (suite *InterpolateTestSuite) TestInvalidError() {
	actual, err := Interpolate("$foo", suite.VariableDictionary)
	if assert.NoError(suite.T (), err) {
		assert.Equal (suite.T (), actual, "$foo")
	}
}

func (suite *InterpolateTestSuite) TestInvalidErrorStrict() {
	_, err := StrictInterpolate("$foo", suite.VariableDictionary)
	assert.EqualError(suite.T (), err, "Invalid `$` sequence \"foo\" found.")
}

func (suite *InterpolateTestSuite) TestRecursion() {
	_, err := Interpolate("${rec}", suite.VariableDictionary)
	assert.EqualError(suite.T(), err, "Recursion limit exceeded.")
}

func (suite *InterpolateTestSuite) TestNonExistent() {
	actual, err := Interpolate("${mokeke}moke", suite.VariableDictionary)
	if assert.NoError(suite.T(), err) {
		assert.Equal(suite.T(), actual, "moke")
	}
}

func (suite *InterpolateTestSuite) TestNonExistentWithStrict () {
	_, err := StrictInterpolate("${mokeke}moke", suite.VariableDictionary)
	assert.EqualError(suite.T(), err, "Unknown reference ${mokeke} found.")
}

func TestInterpolateSuite(t *testing.T) {
	suite.Run(t, new(InterpolateTestSuite))
}
