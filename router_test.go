package main

import (
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"testing"
)

type RouterTestSuite struct {
	suite.Suite
	assert *require.Assertions
	router *mux.Router
}

func TestRouter(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}

func (suite *RouterTestSuite) SetupTest() {
	setContext()
	daemonContext.writePIDFile()
	suite.router = newRouter()
}

func (suite *RouterTestSuite) TearDownTest() {
	daemonContext.removePIDFile()
}

func (suite *RouterTestSuite) TestRouter_Success() {
	req, err := http.NewRequest("GET", "/status", nil)
	suite.Nil(err)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	suite.Equal(http.StatusOK, resp.Code)
}
