package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAdminSuiteEndpointsRejectInvalidIdentity(t *testing.T) {
	for _, handler := range []gin.HandlerFunc{ListAdminEcomSuiteTasks, GetAdminEcomSuiteTask} {
		for _, identity := range []interface{}{nil, "1", uint(0)} {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/admin/ecom-suite/tasks", nil)
			if identity != nil {
				c.Set("userID", identity)
			}
			handler(c)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("invalid identity %v returned %d", identity, w.Code)
			}
		}
	}
}
