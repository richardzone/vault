package http

import (
	"encoding/hex"
	"net/http"
	"reflect"
	"testing"

	"github.com/hashicorp/vault/vault"
)

func TestSysRekeyInit_Status(t *testing.T) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp, err := http.Get(addr + "/v1/sys/rekey/init")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var actual map[string]interface{}
	expected := map[string]interface{}{
		"started":          false,
		"t":                float64(0),
		"n":                float64(0),
		"progress":         float64(0),
		"required":         float64(1),
		"pgp_fingerprints": interface{}(nil),
		"backup":           false,
	}
	testResponseStatus(t, resp, 200)
	testResponseBody(t, resp, &actual)
	expected["nonce"] = actual["nonce"]
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("\nexpected: %#v\nactual: %#v", expected, actual)
	}
}

func TestSysRekeyInit_Setup(t *testing.T) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp := testHttpPut(t, token, addr+"/v1/sys/rekey/init", map[string]interface{}{
		"secret_shares":    5,
		"secret_threshold": 3,
	})
	testResponseStatus(t, resp, 204)

	resp = testHttpGet(t, token, addr+"/v1/sys/rekey/init")

	var actual map[string]interface{}
	expected := map[string]interface{}{
		"started":          true,
		"t":                float64(3),
		"n":                float64(5),
		"progress":         float64(0),
		"required":         float64(1),
		"pgp_fingerprints": interface{}(nil),
		"backup":           false,
	}
	testResponseStatus(t, resp, 200)
	testResponseBody(t, resp, &actual)
	expected["nonce"] = actual["nonce"]
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("\nexpected: %#v\nactual: %#v", expected, actual)
	}
}

func TestSysRekeyInit_Cancel(t *testing.T) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp := testHttpPut(t, token, addr+"/v1/sys/rekey/init", map[string]interface{}{
		"secret_shares":    5,
		"secret_threshold": 3,
	})
	testResponseStatus(t, resp, 204)

	resp = testHttpDelete(t, token, addr+"/v1/sys/rekey/init")
	testResponseStatus(t, resp, 204)

	resp, err := http.Get(addr + "/v1/sys/rekey/init")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var actual map[string]interface{}
	expected := map[string]interface{}{
		"started":          false,
		"t":                float64(0),
		"n":                float64(0),
		"progress":         float64(0),
		"required":         float64(1),
		"pgp_fingerprints": interface{}(nil),
		"backup":           false,
	}
	testResponseStatus(t, resp, 200)
	testResponseBody(t, resp, &actual)
	expected["nonce"] = actual["nonce"]
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("\nexpected: %#v\nactual: %#v", expected, actual)
	}
}

func TestSysRekey_badKey(t *testing.T) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp := testHttpPut(t, token, addr+"/v1/sys/rekey/update", map[string]interface{}{
		"key": "0123",
	})
	testResponseStatus(t, resp, 400)
}

func TestSysRekey_Update(t *testing.T) {
	core, master, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp := testHttpPut(t, token, addr+"/v1/sys/rekey/init", map[string]interface{}{
		"secret_shares":    5,
		"secret_threshold": 3,
	})
	testResponseStatus(t, resp, 204)

	// We need to get the nonce first before we update
	resp, err := http.Get(addr + "/v1/sys/rekey/init")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	var rekeyStatus map[string]interface{}
	testResponseStatus(t, resp, 200)
	testResponseBody(t, resp, &rekeyStatus)

	resp = testHttpPut(t, token, addr+"/v1/sys/rekey/update", map[string]interface{}{
		"nonce": rekeyStatus["nonce"].(string),
		"key":   hex.EncodeToString(master),
	})

	var actual map[string]interface{}
	expected := map[string]interface{}{
		"complete":         true,
		"nonce":            rekeyStatus["nonce"].(string),
		"backup":           false,
		"pgp_fingerprints": interface{}(nil),
	}
	testResponseStatus(t, resp, 200)
	testResponseBody(t, resp, &actual)

	keys := actual["keys"].([]interface{})
	if len(keys) != 5 {
		t.Fatalf("bad: %#v", keys)
	}

	delete(actual, "keys")
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("\nexpected: %#v\nactual: %#v", expected, actual)
	}
}

func TestSysRekey_ReInitUpdate(t *testing.T) {
	core, master, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()
	TestServerAuth(t, addr, token)

	resp := testHttpPut(t, token, addr+"/v1/sys/rekey/init", map[string]interface{}{
		"secret_shares":    5,
		"secret_threshold": 3,
	})
	testResponseStatus(t, resp, 204)

	resp = testHttpDelete(t, token, addr+"/v1/sys/rekey/init")
	testResponseStatus(t, resp, 204)

	resp = testHttpPut(t, token, addr+"/v1/sys/rekey/init", map[string]interface{}{
		"secret_shares":    5,
		"secret_threshold": 3,
	})
	testResponseStatus(t, resp, 204)

	resp = testHttpPut(t, token, addr+"/v1/sys/rekey/update", map[string]interface{}{
		"key": hex.EncodeToString(master),
	})

	testResponseStatus(t, resp, 400)
}
