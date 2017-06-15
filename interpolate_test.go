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
	suite.VariableDictionary = map[string]string{"foo": "foo-value", "bar": "bar-value, ${foo}", "baz": "baz-value, ${bar}"}
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
	assert.Error(suite.T(), err)
}

func (suite *InterpolateTestSuite) TestInvalidError() {
	_, err := Interpolate("$foo", suite.VariableDictionary)
	assert.Error(suite.T(), err)
}

func TestInterpolateSuite(t *testing.T) {
	suite.Run(t, new(InterpolateTestSuite))
}
