package oidc

import (
	"bytes"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"hash/fnv"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/context"

	jose "gopkg.in/square/go-jose.v1"
)

func loadKeySet(t *testing.T, keyfiles ...string) jose.JsonWebKeySet {
	var set jose.JsonWebKeySet
	set.Keys = make([]jose.JsonWebKey, len(keyfiles))
	for i, keyfile := range keyfiles {
		set.Keys[i], _ = loadKey(t, keyfile)
	}
	return set
}

func loadKey(t *testing.T, keyfile string) (pub, priv jose.JsonWebKey) {
	data, err := ioutil.ReadFile(keyfile)
	if err != nil {
		t.Fatalf("can't read key file %s: %v", keyfile, err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatalf("no PEM data found in key file: %s", keyfile)
	}

	keyID := hex.EncodeToString(fnv.New64().Sum(block.Bytes))

	if strings.HasPrefix(filepath.Base(keyfile), "ecdsa") {
		p, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			t.Fatalf("failed to parse ecdsa key %s: %v", keyfile, err)
		}
		priv = jose.JsonWebKey{Key: p, Algorithm: "EC", Use: "sig", KeyID: keyID}
		pub = jose.JsonWebKey{Key: p.Public(), Algorithm: "EC", Use: "sig", KeyID: keyID}
	} else {
		p, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			t.Fatalf("failed to parse rsa key %s: %v", keyfile, err)
		}
		priv = jose.JsonWebKey{Key: p, Algorithm: "RSA", Use: "sig", KeyID: keyID}
		pub = jose.JsonWebKey{Key: p.Public(), Algorithm: "RSA", Use: "sig", KeyID: keyID}
	}
	return
}

func signPayload(t *testing.T, keyfile string, payload []byte) string {
	_, key := loadKey(t, keyfile)
	var (
		signer jose.Signer
		err    error
	)
	if strings.HasPrefix(filepath.Base(keyfile), "rsa") {
		signer, err = jose.NewSigner(jose.RS512, &key)
	} else {
		signer, err = jose.NewSigner(jose.ES512, &key)
	}
	if err != nil {
		t.Fatalf("failed to create signer for %s: %v", keyfile, err)
	}
	jws, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("failed to sign payload: %v", err)
	}
	s, err := jws.CompactSerialize()
	if err != nil {
		t.Fatalf("failed to serialize signature: %v", err)
	}
	return s
}

func TestVerify(t *testing.T) {
	jwks := loadKeySet(t,
		"testdata/ecdsa_521_1.pem",
		"testdata/rsa_2048_1.pem",
	)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(jwks); err != nil {
			t.Errorf("failed to encode jwks: %v", err)
		}
	}))
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := newRemoteKeySet(ctx, s.URL)

	tests := []struct {
		key     string
		wantErr bool
	}{
		{"testdata/ecdsa_521_1.pem", false},
		{"testdata/rsa_2048_1.pem", false},
		{"testdata/ecdsa_521_2.pem", true},
		{"testdata/rsa_2048_2.pem", true},
		{"testdata/ecdsa_521_1.pem", false},
		{"testdata/rsa_2048_1.pem", false},
	}

	for _, tc := range tests {
		data := []byte("foobar")
		payload, err := r.verifyJWT(signPayload(t, tc.key, data))
		if err != nil {
			if !tc.wantErr {
				t.Fatalf("failed to verify JWT signed by %s: %v", tc.key, err)
			}
			continue
		}

		if tc.wantErr {
			t.Fatalf("didn't expecte to be able to verify payload with %s", tc.key)
		}

		if bytes.Compare(payload, data) != 0 {
			t.Errorf("want %q got %q", data, payload)
		}
	}
}

func TestCacheControl(t *testing.T) {
	tests := []struct {
		transport      http.RoundTripper
		expectedMaxAge time.Duration
		url            string
		nKeys          int
	}{
		{
			transport:      new(googleKeysRoundTripper),
			expectedMaxAge: 18213 * time.Second,
			url:            "https://googleapis.com/oauth2/v3/certs",
			nKeys:          4,
		},
		{
			// SalesForce insists on not caching keys.
			transport:      new(salesForceKeysRoundTripper),
			expectedMaxAge: minCache,
			url:            "https://login.salesforce.com/id/keys",
			nKeys:          8,
		},
	}
	for _, tc := range tests {
		client := &http.Client{Transport: tc.transport}

		before := time.Now().Add(tc.expectedMaxAge)
		keys, expiry, err := requestKeys(client, tc.url)
		if err != nil {
			t.Fatalf("Request keys failed: %v", err)
		}
		tAfter := time.Now()
		after := tAfter.Add(tc.expectedMaxAge)
		approxExpiration := expiry.Sub(tAfter)

		if expiry.Before(before) || expiry.After(after) {
			t.Errorf("expected keys to expire in %s: got about %s", tc.expectedMaxAge, approxExpiration)
		}
		if len(keys) != tc.nKeys {
			t.Errorf("expected %d keys got %d", tc.nKeys, len(keys))
		}
	}
}

var googleKeysReq = http.Request{
	Method:     "GET",
	URL:        &url.URL{Scheme: "https", Host: "www.googleapis.com", Path: "/oauth2/v3/certs"},
	Proto:      "HTTP/1.1",
	ProtoMajor: 1,
	ProtoMinor: 1,
	Header: http.Header{
		"Host":       []string{"www.googleapis.com"},
		"User-Agent": []string{"curl/7.43.0"},
		"Accept":     []string{"*/*"},
	},
	Host: "www.googleapis.com",
}

type googleKeysRoundTripper struct{}

func (_ *googleKeysRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"Expires":                []string{"Sat, 16 Jul 2016 01:27:10 GMT"},
			"Date":                   []string{"Fri, 15 Jul 2016 20:23:37 GMT"},
			"Vary":                   []string{"Origin", "X-Origin"},
			"Content-Type":           []string{"application/json; charset=UTF-8"},
			"X-Content-Type-Options": []string{"nosniff"},
			"X-Frame-Options":        []string{"SAMEORIGIN"},
			"X-XSS-Protection":       []string{"1; mode=block"},
			"Content-Length":         []string{"1957"},
			"Server":                 []string{"GSE"},
			"Cache-Control":          []string{"public, max-age=18213, must-revalidate, no-transform"},
			"Age":                    []string{"13156"},
			"Alternate-Protocol":     []string{"443:quic"},
			"Alt-Svc":                []string{`quic=":443"; ma=2592000; v="36,35,34,33,32,31,30,29,28,27,26,25"`},
		},
		ContentLength: 1957,
		Body:          ioutil.NopCloser(strings.NewReader(googleKeysBody)),
	}, nil
}

type salesForceKeysRoundTripper struct{}

func (_ *salesForceKeysRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"Date": []string{"Mon, 18 Jul 2016 23:06:26 GMT"},
			"Strict-Transport-Security":           []string{"max-age=10886400; includeSubDomains; preload"},
			"Content-Security-Policy-Report-Only": []string{"default-src https:; script-src https: 'unsafe-inline' 'unsafe-eval'; style-src https: 'unsafe-inline'; img-src https: data:; font-src https: data:; report-uri /_/ContentDomainCSPNoAuth?type=login"},
			"Set-Cookie":                          []string{"BrowserId=YQwZPagMRDiCEqn6s7D4cg;Path=/;Domain=.salesforce.com;Expires=Fri, 16-Sep-2016 23:06:26 GMT"},
			"Expires":                             []string{"Thu, 01 Jan 1970 00:00:00 GMT"},
			"Content-Type":                        []string{"application/json;charset=UTF-8"},
			"Pragma":                              []string{"no-cache"},
			"Cache-Control":                       []string{"no-cache, no-store"},
			"Transfer-Encoding":                   []string{"chunked"},
		},
		ContentLength: -1,
		Body:          ioutil.NopCloser(strings.NewReader(salesForceKeysBody)),
	}, nil
}

var googleKeysBody = `{
 "keys": [
  {
   "kty": "RSA",
   "alg": "RS256",
   "use": "sig",
   "kid": "40aa42edac0614d7ca3f57f97ee866cdfba3b61a",
   "n": "6lm9AEGLPFpVqnfeVFuTIZsj7vz_kxla6uW1WWtosM_MtIjXkyyiSolxiSOs3bzG66iVm71023QyOzKYFbio0hI-yZauG3g9nH-zb_AHScsjAKagHtrHmTdtq0JcNkQnAaaUwxVbjwMlYAcOh87W5jWj_MAcPvc-qjy8-WJ81UgoOUZNiKByuF4-9igxKZeskGRXuTPX64kWGBmKl-tM7VnCGMKoK3m92NPrktfBoNN_EGGthNfQsKFUdQFJFtpMuiXp9Gib7dcMGabxcG2GUl-PU086kPUyUdUYiMN2auKSOxSUZgDjT7DcI8Sn8kdQ0-tImaHi54JNa1PNNdKRpw",
   "e": "AQAB"
  },
  {
   "kty": "RSA",
   "alg": "RS256",
   "use": "sig",
   "kid": "8fbbeea40332d2c0d27e37e1904af29b64594e57",
   "n": "z7h6_rt35-j6NV2iQvYIuR3xvsxmEImgMl8dc8CFl4SzEWrry3QILajKxQZA9YYYfXIcZUG_6R6AghVMJetNIl2AhCoEr3RQjjNsm9PE6h5p2kQ-zIveFeb__4oIkVihYtxtoYBSdVj69nXLUAJP2bxPfU8RDp5X7hT62pKR05H8QLxH8siIQ5qR2LGFw_dJcitAVRRQofuaj_9u0CLZBfinqyRkBc7a0zi7pBxtEiIbn9sRr8Kkb_Boap6BHbnLS-YFBVarcgFBbifRf7NlK5dqE9z4OUb-dx8wCMRIPVAx_hV4Qx2anTgp1sDA6V4vd4NaCOZX-mSctNZqQmKtNw",
   "e": "AQAB"
  },
  {
   "kty": "RSA",
   "alg": "RS256",
   "use": "sig",
   "kid": "6758b0b8eb341e90454860432d6a1648bf4de03b",
   "n": "5K0rYaA7xtqSe1nFn_nCA10uUXY81NcohMeFsYLbBlx_NdpsmbpgtXJ6ektYR7rUdtMMLu2IONlNhkWlx-lge91okyacUrWHP88PycilUE-RnyVjbPEm3seR0VefgALfN4y_e77ljq2F7W2_kbUkTvDzriDIWvQT0WwVF5FIOBydfDDs92S-queaKgLBwt50SXJCZryLew5ODrwVsFGI4Et6MLqjS-cgWpCNwzcRqjBRsse6DXnex_zSRII4ODzKIfX4qdFBKZHO_BkTsK9DNkUayrr9cz8rFRK6TEH6XTVabgsyd6LP6PTxhpiII_pTYRSWk7CGMnm2nO0dKxzaFQ",
   "e": "AQAB"
  },
  {
   "kty": "RSA",
   "alg": "RS256",
   "use": "sig",
   "kid": "0b915c6a65e2651ddcd0977757ebc644220a23b5",
   "n": "vKa1quTzaKez5hyWc1SRYA_pgCEUx2FgpcfHEz33IjruqM57QTg4jZ2sukU5JdkjegtZm_ry9FuFKB-tIFszNqRBc4hgKWNzvaXicPpGP-tWFxhe30esaXUhF4WpZd4uhLfowJxuXKkM0qjRFAbQiP_N64fauozjquLfESaT0WdclK-wACzb-Mo9GoxYzgmPSRvmNJ83ZfBOimcIfuCQZkIFUjHrYRJ-kZfS02_tkkqpP8KaU3QL0igsQglawJpvH3TTZbA0xuJMOwGESN1xq4Xr_7o3OfNTTfybdVv1o8FMT8snIATGwKXvi9J3P7OEV6r8_pdFUhCOAvRlUYw5-Q",
   "e": "AQAB"
  }
 ]
}`

var salesForceKeysBody = `{"keys":[{"kty":"RSA","n":"nOcQOvHV8rc-hcfP_RmxMGjyVlruSLeFXTojYcbixaAH36scUejjaws31orUjmYqB5isE9ntdsL4DnsdP_MDJ2mtYD2FIh8tBkJjgXitjdcDclrwELAx846wBIlSES8wR6czpdJZfSwhL_92EGpDH6z7lKEClqhDlbtZ-yFKFj9BQRwaEXWV7uuq23gxXOqyEN0WXl3ZJPgsodCnlXRn9y_r5CNV9V4wvzXGlJhT3Nv_N_Z5XNZIjZnHdCuE_itT4a1xENEEds7Jjg5mRTlVFzYv5iQtBo7jdY5ogMTgKPmRh6hYuqLeki3AOAUff1AGaN9TZH60UxwTw03-DQJL5C2SuC_vM5KIWxZxQniubfegUCBXpJSAJbLt8zSFztTcrLS4-wgUHo1A8TDNaO28_KsBUTWsrieOr3NfCn4bPNb7t8G90U60lW0GIhEda3fNYnV0WWpZVO1jCRNy_JYUs3ECo0E1ZQJZD72Dm6UjiuH7eR3ZgNKR9tlLNdyZSpZUZPErLrXJ90d5XbmJYvRX9r93z6GQqOv5FQy1JhatwefxhKdyhkDEHsqELO0XDqnDnmgxkEEU-lHYSVGz-iDlUZOUYTTCtxsPDmBIXOMuwp0UydJphO36qRQaDyEjHNsYKLj5KVvjDHS8Gw1FhbFvsoUrBHre4hLY9Pa5meatV_k","e":"AQAB","alg":"RS256","use":"sig","kid":"192"},{"kty":"RSA","n":"sTuRuR0_kfPcrMug1dg4e3y4_elyaKDyzqlvEpzo6FgYTAuR-G9wv4Au6O363PDF8Yd8wg5JSwLjcHlfp3auXb_9pzJyhMBUAQwUYhVOPyxGOiyGYwCwkUX_GUA8IGPQZ6XOISX65492upv2DXW85vxYjPGJypWIQ3GjYrYtqmEg2fkuXX3IwITMxHoxYSmvR5Qcn4acnxRKMqCohxLANDEN4xyZBGbuchXnb0TGFwcGNAO8mz-O4E_A9k3QQwz750Qz6xEe8_UWPRvyX32EvGN19tX90DcT488YT0ImdUt6sVZEyOU_vxYXFFpWhf2_C1_AcjUQejex5y4mQn3idtVSAVGQ_eqhS3ZpyY_IhAGSC1JR_TJfrgnJoQrvv7Toz4K-iskOkFDNZYjhkKSla0EaJOUjRdZxmFTlzydmJFCD16-dbB8cCYGEQWVhnmBnYRQ2zquTCGReuILEcV0ALo9sR7zA5u8xSoJpZO1-KM_9lfbBH5M8V6R-ECWJI5MqPJxK0D3BCiNNxI5Kdslb_49SNA3qukQciassa-xbm26MGyYjt7Kn2KC2Tdh8L0z3328Pe6Wn7-vixSbtSc_qy15OifYKHr7BE1JjZfbkFQ6ZY5WgHZOZWG31-9aIhKsCKJGfGwyaN-I2-IzygSIihJNrFnSrO_s80li3H8P-4A0","e":"AQAB","alg":"RS256","use":"sig","kid":"198"},{"kty":"RSA","n":"n9ggEbWxKGy-cke0GpNQlblk47vL7-rtSsPwehgCdt7vANf6n6eqopLN1WPfq-xRJo7aE4gBXV3hQ4Ts1vm3_lXZmUCcFl8iVEYtDUZakmCYlDZExpo5AuRpwIayaidOzFP2qpMNstd9iVN_0ShCNjN-I_kJXAliqxuEZw5A-P-mcUBkE5ar_xVV3TaAXsxU5cecou9fbQjSLSKpQzfKVdUr0japx4iwSv0vdjXzItUfu7CykHU03Xmd4ucv47gdE0FeESgtPvC6cz6W6qCddP3rjBlgYAg0xYu_uyfLLYyaZZoEx8PfYUXfC4zM2a9p7bT1RxfRJDIqA5S507CylpzeAdBdp0N9UCtecENVgVcRotFBv1GOuVEhkba93spdddCGTSjWqzbq2hVj4napeGZ3I1st0mLQCD2reUh6injMFkVllnpfOitjvM98V-c_cxiTJYfg8K338atpWM7nR9Yo5D684Pr-z77xTl0_WY8mIukmGQtznX2dEYYbyzj_PLtK4YSPyrvnqdGFcZpxstSWOHnCeLNigzWCCBbcO2B8F3QGDagW6Zu7IH_P3KRPqwKWDzGzpmDJPGYexz46qezjVnPm-aCcx_m6cnqXqMMt60YtCMFCLxyyBJrTx66LNUpxVRun2vAu2HHdwxv1lzrEvDMzwrSW9JZVmAJ_eg0","e":"AQAB","alg":"RS256","use":"sig","kid":"202"},{"kty":"RSA","n":"o8Fz0jXjZ0Rz5Kt2TmzP0xVokf-Q4Az-MQg5i5MCxNNTQiZp7VkwAZeM0mJ-mKDbCzPm9ws43v8cxeiIkVZQqrAocnnb90MDCnU-7oD7MvOU4SbmhuLzVCyVZPIBRq5z0OgjcwLeD4trOoogkLOu0kyuyzNoFkr712m_GZ1xic-X0MlFKq3-2cI4U2nEuuh-Xcy7bUqCx0zTJFPOOKghGYEZZ6biZ04VC-ERcW6cC19pEWm6vCqZJEsKPCfazVAoHKZAukNd0XLPQd_W6xAaGnp8e7a5tFHn6dU6ikhI94ZieVp6WItWsQTDwJH-D7bVpVRG-lWL74lgcuQdFAtldm__k7FvlTXdqiLrd0rYuDnTFiwUSsUXWBJbmGVsEOylZVPQAL-K7G7p3BRY4X26vOgfludwCOj7L7WFbd0IXziTm74xe2KZGKsFpoCjJI0z_D5Oe5bofswr1Ceafhl97suG7OoInobt7QAQnnLcBVzUPz_TflOXDc5UiePptA0bxdd8MVENiDbTGGNz6DCzfL986QfcJLPB8aZa3lFN0kWPBkOclZagL4WpyIllB6euvZ2hfpt8IY2_bmUN06luo6N7Fy0hSSFMWvfzaD8_Ff3czb1Kv-b0xI6Ugk4d67RNNSbTcRM2Muvx-dJgOyXqrc_hE96OOqcMjrGZJoXnCAM","e":"AQAB","alg":"RS256","use":"sig","kid":"190"},{"kty":"RSA","n":"wIQtK09qsu1qCCQu1mHh6d_EyyOlbqMCV8WMacOyhZng1sbaFJY-0PIH46Kw1uhjbHg94_r2UELYd30vF8xwViGhCmpPuSGhkxNoT5CMoJPS6JW-zBpR7suHqBUnaGdZ6G2uYZDpwWYs_4SJDuWzxVBrQqIM_ZVgUqutniQPmjMAX5MqznBTnG3zm728BmNzS7T2gtzxs3jAgDsSAu3Kxp3D6NDGERhaAJ8jOgwHvmQK5xFi9Adw7sv2nCH-wM-C5fLJYmpGOSrTP1HLOlq--TROAvWL9gcNEeq4arryIYux5syg66rHT8U2Uhb1PdXt7ReQY8wBnP2BBH1QH7rzOZ7UbqFLbQUQsZFAVMcfm7gJN8JWLlcSJZdC2zaY0wI5q8PWN-N_GgAK64FKZQ7pB0bRQ5AQx-D3U4sYE4EcgSvV8fW86PaF1VXaHMFcom48gZ1GzE_V25uPb-0yue0cv9lejrIKDvRiJ5UiyUPphro4Aw2ZcDi_8r8rqfglWhcnB4bGSri4kEBb_IdwvqKwRCqxlNdRnU1ooQeUBaVRwdbpj23Z1qtYjB55Wf2KOCJ6ewMyddq4bEAG6KIqPmssT7_exvygUyuW6qhnCV-gTZEwFI0A6djsHM5itfkzNY47BeuAtGXjuaRnVYIEvTrnSj3Lx7YfvCIiGqFrG6y31Ak","e":"AQAB","alg":"RS256","use":"sig","kid":"188"},{"kty":"RSA","n":"hsqiqMXZmxJHzWfZwbSffKfc9YYMxj83-aWhA91jtI8k-GMsEB6mtoNWLP6vmz6x6BQ8Sn6kmn65n1IGCIlWxhPn9yqfXBDBaHFGYED9bBloSEMFnnS9-ACsWrHl5UtDQ3nh-VQTKg1LBmjJMmAOHdBLoUikfpx8fjA1LfDn_1iNWnguj2ehgjWCuTn64UdUd84YNcfO8Ha0TAhWHOhkiluMyzGS0dtN0h8Ybyi5oL6Bf1sfhtOncUh1JuWMcmvICbGEkA_0vBbMp9nCvXdMlpzMOCIoYYkQ-25SRZ0GpIr_oBIZByEm1XaJIqNXoC7qJ95iAyWkUiSegY_IcBV3nMXr-kDNn9Vm2cgLEJGymOiDQKH8g7VjraCIrqWPD3DWv3Z6RsExs6i0gG3JU9cVVFwz87d05_yk3L5ubWb96uxsP9rkwZ3h8eJTfFrgMhk1ZwR-63Dk3ZLYisiAU0zKgr4vQ9qsCNPqDg0rkeqOY5k7Gy201_wh6Sw5dCNTTGmZZ1rNE-gyDu4-a1H40n8f2JFiH-xIOD9-w8HGYOu_oGlobK2KvzFYHTk-w7vtfhZ0j96UkjaBhVjYSMi4hf43xNbB4xJoHhHLESABLp9IYDlnzBeBXKumXDO5aRk3sFAEAWxj57Ec_DyK6UwXSR9Xqji5a1lEArUdFPYzVZ_YCec","e":"AQAB","alg":"RS256","use":"sig","kid":"194"},{"kty":"RSA","n":"5SGw1jcqyFYEZaf39RoxAhlq-hfRSOsneVtsT2k09yEQhwB2myvf3ckVAwFyBF6y0Hr1psvu1FlPzKQ9YfcQkfge4e7eeQ7uaez9mMQ8RpyAFZprq1iFCix4XQw-jKW47LAevr9w1ttZY932gFrGJ4gkf_uqutUny82vupVUETpQ6HDmIL958SxYb_-d436zi5LMlHnTxcR5TWIQGGxip-CrD7vOA3hrssYLhNGQdwVYtwI768EvwE8h4VJDgIrovoHPH1ofDQk8-oG20eEmZeWugI1K3z33fZJS-E_2p_OiDVr0EmgFMTvPTnQ75h_9vyF1qhzikJpN9P8KcEm8oGu7KJGIn8ggUY0ftqKG2KcWTaKiirFFYQ981PhLHryH18eOIxMpoh9pRXf2y7DfNTyid99ig0GUH-lzAlbKY0EV2sIuvEsIoo6G8YT2uI72xzl7sCcp41FS7oFwbUyHp_uHGiTZgN7g-18nm2TFmQ_wGB1xCwJMFzjIXq1PwEjmg3W5NBuMLSbG-aDwjeNrcD_4vfB6yg548GztQO2MpV_BuxtrZDJQm-xhJXdm4FfrJzWdwX_JN9qfsP0YU1_mxtSU_m6EKgmwFdE3Yh1WM0-kRRSk3gmNvXpiKeVduzm8I5_Jl7kwLgBw24QUVaLZn8jC2xWRk_jcBNFFLQgOf9U","e":"AQAB","alg":"RS256","use":"sig","kid":"196"},{"kty":"RSA","n":"m-rZsEmySPnZLZ9tdoQ9bW0sI_nudwy70vAXK1JZV9AhCJQB5ZHykaK90mpmwAdt8XsrOuQ6Nd9hrgZOHgq-RznNCFhE9qpnOHg68ywsUeZHXD9FqX_QlKPzMBBQXhyWLz58_LHMmGhn4740rB7tXftvDBOctX41I_hilm8_vKbSPE2ov1h9dYMt0a9jFTd6xk5dnj_r3KiswIl4FG4KoXHCv7Hzgv6iVyOJJgTgJcmRw9ydDxvEzkyXlIYFsOW0IKxAoA28ECAImPmiRX37-oP_IK6ZrjxNHd7SqQS12uc33N1ZfI3WR35GGWfLTH7IjjLj0c8lvUrQOQTZc686wqySRZz-BWSFBAYR8OLA-T1SZVk3R_FyZ26mXNYo33I9DcuK5y6c_AMoO0c4Mw-WmTkrAu5QTPWJ-iQsbMjfIDC7XkxkBoIMqIzynZoV1fH5yO_2CyWQfsr1PJRjp3tFhcVgUBrhVY47IsDQ9pmE8XgtRe9dQGT2wXn-aANqF6vQkw_kJ-zDD5VWQ_IQol_znwyktAoYB7iRwS2Ut4l4YsFE5D9SywHK0F1mvIQWENNbw2WjIxgjI8DtJFPdTN_dhnZDfkKjsxvQlchIiMZGAErhaveyyx4ezbUo4XAtzj84QAj9IjtEjOkEo6MqDDeCh6pXlIV_UBdblTXoX37DRpc","e":"AQAB","alg":"RS256","use":"sig","kid":"200"}]}`
