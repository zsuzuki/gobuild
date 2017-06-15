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

func (suite *InterpolateTestSuite) SetupTest () {
	suite.VariableDictionary = map[string]string { "foo" : "foo-value", "bar" : "bar-value", "baz" : "baz-value" }
}

// Interpolate empty string
func (suite *InterpolateTestSuite) TestEmpty () {
	actual, err := Interpolate("", suite.VariableDictionary)
	if assert.NoError(suite.T(), err) {
		assert.Equal(suite.T(), "", actual)
	}
}

// Interpolate literal
func (suite *InterpolateTestSuite) TestLiteral () {
	actual, err := Interpolate("literal", suite.VariableDictionary)
	if assert.NoError(suite.T (), err) {
		assert.Equal(suite.T (), "literal", actual)
	}
}

func (suite *InterpolateTestSuite) TestSimpleSubstitution () {
	actual, err := Interpolate("${foo}", suite.VariableDictionary)
	if assert.NoError(suite.T(), err) {
		assert.Equal(suite.T(), "foo-value", actual)
	}
}

func TestInterpolateSuite(t *testing.T) {
	suite.Run (t, new (InterpolateTestSuite))
}
