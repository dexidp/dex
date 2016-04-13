// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webdav

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestReadLockInfo(t *testing.T) {
	// The "section x.y.z" test cases come from section x.y.z of the spec at
	// http://www.webdav.org/specs/rfc4918.html
	testCases := []struct {
		desc       string
		input      string
		wantLI     lockInfo
		wantStatus int
	}{{
		"bad: junk",
		"xxx",
		lockInfo{},
		http.StatusBadRequest,
	}, {
		"bad: invalid owner XML",
		"" +
			"<D:lockinfo xmlns:D='DAV:'>\n" +
			"  <D:lockscope><D:exclusive/></D:lockscope>\n" +
			"  <D:locktype><D:write/></D:locktype>\n" +
			"  <D:owner>\n" +
			"    <D:href>   no end tag   \n" +
			"  </D:owner>\n" +
			"</D:lockinfo>",
		lockInfo{},
		http.StatusBadRequest,
	}, {
		"bad: invalid UTF-8",
		"" +
			"<D:lockinfo xmlns:D='DAV:'>\n" +
			"  <D:lockscope><D:exclusive/></D:lockscope>\n" +
			"  <D:locktype><D:write/></D:locktype>\n" +
			"  <D:owner>\n" +
			"    <D:href>   \xff   </D:href>\n" +
			"  </D:owner>\n" +
			"</D:lockinfo>",
		lockInfo{},
		http.StatusBadRequest,
	}, {
		"bad: unfinished XML #1",
		"" +
			"<D:lockinfo xmlns:D='DAV:'>\n" +
			"  <D:lockscope><D:exclusive/></D:lockscope>\n" +
			"  <D:locktype><D:write/></D:locktype>\n",
		lockInfo{},
		http.StatusBadRequest,
	}, {
		"bad: unfinished XML #2",
		"" +
			"<D:lockinfo xmlns:D='DAV:'>\n" +
			"  <D:lockscope><D:exclusive/></D:lockscope>\n" +
			"  <D:locktype><D:write/></D:locktype>\n" +
			"  <D:owner>\n",
		lockInfo{},
		http.StatusBadRequest,
	}, {
		"good: empty",
		"",
		lockInfo{},
		0,
	}, {
		"good: plain-text owner",
		"" +
			"<D:lockinfo xmlns:D='DAV:'>\n" +
			"  <D:lockscope><D:exclusive/></D:lockscope>\n" +
			"  <D:locktype><D:write/></D:locktype>\n" +
			"  <D:owner>gopher</D:owner>\n" +
			"</D:lockinfo>",
		lockInfo{
			XMLName:   xml.Name{Space: "DAV:", Local: "lockinfo"},
			Exclusive: new(struct{}),
			Write:     new(struct{}),
			Owner: owner{
				InnerXML: "gopher",
			},
		},
		0,
	}, {
		"section 9.10.7",
		"" +
			"<D:lockinfo xmlns:D='DAV:'>\n" +
			"  <D:lockscope><D:exclusive/></D:lockscope>\n" +
			"  <D:locktype><D:write/></D:locktype>\n" +
			"  <D:owner>\n" +
			"    <D:href>http://example.org/~ejw/contact.html</D:href>\n" +
			"  </D:owner>\n" +
			"</D:lockinfo>",
		lockInfo{
			XMLName:   xml.Name{Space: "DAV:", Local: "lockinfo"},
			Exclusive: new(struct{}),
			Write:     new(struct{}),
			Owner: owner{
				InnerXML: "\n    <D:href>http://example.org/~ejw/contact.html</D:href>\n  ",
			},
		},
		0,
	}}

	for _, tc := range testCases {
		li, status, err := readLockInfo(strings.NewReader(tc.input))
		if tc.wantStatus != 0 {
			if err == nil {
				t.Errorf("%s: got nil error, want non-nil", tc.desc)
				continue
			}
		} else if err != nil {
			t.Errorf("%s: %v", tc.desc, err)
			continue
		}
		if !reflect.DeepEqual(li, tc.wantLI) || status != tc.wantStatus {
			t.Errorf("%s:\ngot  lockInfo=%v, status=%v\nwant lockInfo=%v, status=%v",
				tc.desc, li, status, tc.wantLI, tc.wantStatus)
			continue
		}
	}
}

func TestReadPropfind(t *testing.T) {
	testCases := []struct {
		desc       string
		input      string
		wantPF     propfind
		wantStatus int
	}{{
		desc: "propfind: propname",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:propname/>\n" +
			"</A:propfind>",
		wantPF: propfind{
			XMLName:  xml.Name{Space: "DAV:", Local: "propfind"},
			Propname: new(struct{}),
		},
	}, {
		desc:  "propfind: empty body means allprop",
		input: "",
		wantPF: propfind{
			Allprop: new(struct{}),
		},
	}, {
		desc: "propfind: allprop",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"   <A:allprop/>\n" +
			"</A:propfind>",
		wantPF: propfind{
			XMLName: xml.Name{Space: "DAV:", Local: "propfind"},
			Allprop: new(struct{}),
		},
	}, {
		desc: "propfind: allprop followed by include",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:allprop/>\n" +
			"  <A:include><A:displayname/></A:include>\n" +
			"</A:propfind>",
		wantPF: propfind{
			XMLName: xml.Name{Space: "DAV:", Local: "propfind"},
			Allprop: new(struct{}),
			Include: propfindProps{xml.Name{Space: "DAV:", Local: "displayname"}},
		},
	}, {
		desc: "propfind: include followed by allprop",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:include><A:displayname/></A:include>\n" +
			"  <A:allprop/>\n" +
			"</A:propfind>",
		wantPF: propfind{
			XMLName: xml.Name{Space: "DAV:", Local: "propfind"},
			Allprop: new(struct{}),
			Include: propfindProps{xml.Name{Space: "DAV:", Local: "displayname"}},
		},
	}, {
		desc: "propfind: propfind",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:prop><A:displayname/></A:prop>\n" +
			"</A:propfind>",
		wantPF: propfind{
			XMLName: xml.Name{Space: "DAV:", Local: "propfind"},
			Prop:    propfindProps{xml.Name{Space: "DAV:", Local: "displayname"}},
		},
	}, {
		desc: "propfind: prop with ignored comments",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:prop>\n" +
			"    <!-- ignore -->\n" +
			"    <A:displayname><!-- ignore --></A:displayname>\n" +
			"  </A:prop>\n" +
			"</A:propfind>",
		wantPF: propfind{
			XMLName: xml.Name{Space: "DAV:", Local: "propfind"},
			Prop:    propfindProps{xml.Name{Space: "DAV:", Local: "displayname"}},
		},
	}, {
		desc: "propfind: propfind with ignored whitespace",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:prop>   <A:displayname/></A:prop>\n" +
			"</A:propfind>",
		wantPF: propfind{
			XMLName: xml.Name{Space: "DAV:", Local: "propfind"},
			Prop:    propfindProps{xml.Name{Space: "DAV:", Local: "displayname"}},
		},
	}, {
		desc: "propfind: propfind with ignored mixed-content",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:prop>foo<A:displayname/>bar</A:prop>\n" +
			"</A:propfind>",
		wantPF: propfind{
			XMLName: xml.Name{Space: "DAV:", Local: "propfind"},
			Prop:    propfindProps{xml.Name{Space: "DAV:", Local: "displayname"}},
		},
	}, {
		desc: "propfind: propname with ignored element (section A.4)",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:propname/>\n" +
			"  <E:leave-out xmlns:E='E:'>*boss*</E:leave-out>\n" +
			"</A:propfind>",
		wantPF: propfind{
			XMLName:  xml.Name{Space: "DAV:", Local: "propfind"},
			Propname: new(struct{}),
		},
	}, {
		desc:       "propfind: bad: junk",
		input:      "xxx",
		wantStatus: http.StatusBadRequest,
	}, {
		desc: "propfind: bad: propname and allprop (section A.3)",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:propname/>" +
			"  <A:allprop/>" +
			"</A:propfind>",
		wantStatus: http.StatusBadRequest,
	}, {
		desc: "propfind: bad: propname and prop",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:prop><A:displayname/></A:prop>\n" +
			"  <A:propname/>\n" +
			"</A:propfind>",
		wantStatus: http.StatusBadRequest,
	}, {
		desc: "propfind: bad: allprop and prop",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:allprop/>\n" +
			"  <A:prop><A:foo/><A:/prop>\n" +
			"</A:propfind>",
		wantStatus: http.StatusBadRequest,
	}, {
		desc: "propfind: bad: empty propfind with ignored element (section A.4)",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <E:expired-props/>\n" +
			"</A:propfind>",
		wantStatus: http.StatusBadRequest,
	}, {
		desc: "propfind: bad: empty prop",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:prop/>\n" +
			"</A:propfind>",
		wantStatus: http.StatusBadRequest,
	}, {
		desc: "propfind: bad: prop with just chardata",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:prop>foo</A:prop>\n" +
			"</A:propfind>",
		wantStatus: http.StatusBadRequest,
	}, {
		desc: "bad: interrupted prop",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:prop><A:foo></A:prop>\n",
		wantStatus: http.StatusBadRequest,
	}, {
		desc: "bad: malformed end element prop",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:prop><A:foo/></A:bar></A:prop>\n",
		wantStatus: http.StatusBadRequest,
	}, {
		desc: "propfind: bad: property with chardata value",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:prop><A:foo>bar</A:foo></A:prop>\n" +
			"</A:propfind>",
		wantStatus: http.StatusBadRequest,
	}, {
		desc: "propfind: bad: property with whitespace value",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:prop><A:foo> </A:foo></A:prop>\n" +
			"</A:propfind>",
		wantStatus: http.StatusBadRequest,
	}, {
		desc: "propfind: bad: include without allprop",
		input: "" +
			"<A:propfind xmlns:A='DAV:'>\n" +
			"  <A:include><A:foo/></A:include>\n" +
			"</A:propfind>",
		wantStatus: http.StatusBadRequest,
	}}

	for _, tc := range testCases {
		pf, status, err := readPropfind(strings.NewReader(tc.input))
		if tc.wantStatus != 0 {
			if err == nil {
				t.Errorf("%s: got nil error, want non-nil", tc.desc)
				continue
			}
		} else if err != nil {
			t.Errorf("%s: %v", tc.desc, err)
			continue
		}
		if !reflect.DeepEqual(pf, tc.wantPF) || status != tc.wantStatus {
			t.Errorf("%s:\ngot  propfind=%v, status=%v\nwant propfind=%v, status=%v",
				tc.desc, pf, status, tc.wantPF, tc.wantStatus)
			continue
		}
	}
}

func TestMultistatusWriter(t *testing.T) {
	if go1Dot4 {
		t.Skip("TestMultistatusWriter requires Go version 1.5 or greater")
	}

	///The "section x.y.z" test cases come from section x.y.z of the spec at
	// http://www.webdav.org/specs/rfc4918.html
	testCases := []struct {
		desc        string
		responses   []response
		respdesc    string
		writeHeader bool
		wantXML     string
		wantCode    int
		wantErr     error
	}{{
		desc: "section 9.2.2 (failed dependency)",
		responses: []response{{
			Href: []string{"http://example.com/foo"},
			Propstat: []propstat{{
				Prop: []Property{{
					XMLName: xml.Name{
						Space: "http://ns.example.com/",
						Local: "Authors",
					},
				}},
				Status: "HTTP/1.1 424 Failed Dependency",
			}, {
				Prop: []Property{{
					XMLName: xml.Name{
						Space: "http://ns.example.com/",
						Local: "Copyright-Owner",
					},
				}},
				Status: "HTTP/1.1 409 Conflict",
			}},
			ResponseDescription: "Copyright Owner cannot be deleted or altered.",
		}},
		wantXML: `` +
			`<?xml version="1.0" encoding="UTF-8"?>` +
			`<multistatus xmlns="DAV:">` +
			`  <response>` +
			`    <href>http://example.com/foo</href>` +
			`    <propstat>` +
			`      <prop>` +
			`        <Authors xmlns="http://ns.example.com/"></Authors>` +
			`      </prop>` +
			`      <status>HTTP/1.1 424 Failed Dependency</status>` +
			`    </propstat>` +
			`    <propstat xmlns="DAV:">` +
			`      <prop>` +
			`        <Copyright-Owner xmlns="http://ns.example.com/"></Copyright-Owner>` +
			`      </prop>` +
			`      <status>HTTP/1.1 409 Conflict</status>` +
			`    </propstat>` +
			`  <responsedescription>Copyright Owner cannot be deleted or altered.</responsedescription>` +
			`</response>` +
			`</multistatus>`,
		wantCode: StatusMulti,
	}, {
		desc: "section 9.6.2 (lock-token-submitted)",
		responses: []response{{
			Href:   []string{"http://example.com/foo"},
			Status: "HTTP/1.1 423 Locked",
			Error: &xmlError{
				InnerXML: []byte(`<lock-token-submitted xmlns="DAV:"/>`),
			},
		}},
		wantXML: `` +
			`<?xml version="1.0" encoding="UTF-8"?>` +
			`<multistatus xmlns="DAV:">` +
			`  <response>` +
			`    <href>http://example.com/foo</href>` +
			`    <status>HTTP/1.1 423 Locked</status>` +
			`    <error><lock-token-submitted xmlns="DAV:"/></error>` +
			`  </response>` +
			`</multistatus>`,
		wantCode: StatusMulti,
	}, {
		desc: "section 9.1.3",
		responses: []response{{
			Href: []string{"http://example.com/foo"},
			Propstat: []propstat{{
				Prop: []Property{{
					XMLName: xml.Name{Space: "http://ns.example.com/boxschema/", Local: "bigbox"},
					InnerXML: []byte(`` +
						`<BoxType xmlns="http://ns.example.com/boxschema/">` +
						`Box type A` +
						`</BoxType>`),
				}, {
					XMLName: xml.Name{Space: "http://ns.example.com/boxschema/", Local: "author"},
					InnerXML: []byte(`` +
						`<Name xmlns="http://ns.example.com/boxschema/">` +
						`J.J. Johnson` +
						`</Name>`),
				}},
				Status: "HTTP/1.1 200 OK",
			}, {
				Prop: []Property{{
					XMLName: xml.Name{Space: "http://ns.example.com/boxschema/", Local: "DingALing"},
				}, {
					XMLName: xml.Name{Space: "http://ns.example.com/boxschema/", Local: "Random"},
				}},
				Status:              "HTTP/1.1 403 Forbidden",
				ResponseDescription: "The user does not have access to the DingALing property.",
			}},
		}},
		respdesc: "There has been an access violation error.",
		wantXML: `` +
			`<?xml version="1.0" encoding="UTF-8"?>` +
			`<multistatus xmlns="DAV:">` +
			`  <response>` +
			`    <href>http://example.com/foo</href>` +
			`    <propstat>` +
			`      <prop>` +
			`        <bigbox xmlns="http://ns.example.com/boxschema/"><BoxType xmlns="http://ns.example.com/boxschema/">Box type A</BoxType></bigbox>` +
			`        <author xmlns="http://ns.example.com/boxschema/"><Name xmlns="http://ns.example.com/boxschema/">J.J. Johnson</Name></author>` +
			`      </prop>` +
			`      <status>HTTP/1.1 200 OK</status>` +
			`    </propstat>` +
			`    <propstat>` +
			`      <prop>` +
			`        <DingALing xmlns="http://ns.example.com/boxschema/"></DingALing>` +
			`        <Random xmlns="http://ns.example.com/boxschema/"></Random>` +
			`      </prop>` +
			`      <status>HTTP/1.1 403 Forbidden</status>` +
			`      <responsedescription>The user does not have access to the DingALing property.</responsedescription>` +
			`    </propstat>` +
			`  </response>` +
			`  <responsedescription>There has been an access violation error.</responsedescription>` +
			`</multistatus>`,
		wantCode: StatusMulti,
	}, {
		desc: "no response written",
		// default of http.responseWriter
		wantCode: http.StatusOK,
	}, {
		desc:     "no response written (with description)",
		respdesc: "too bad",
		// default of http.responseWriter
		wantCode: http.StatusOK,
	}, {
		desc:        "empty multistatus with header",
		writeHeader: true,
		wantXML:     `<multistatus xmlns="DAV:"></multistatus>`,
		wantCode:    StatusMulti,
	}, {
		desc: "bad: no href",
		responses: []response{{
			Propstat: []propstat{{
				Prop: []Property{{
					XMLName: xml.Name{
						Space: "http://example.com/",
						Local: "foo",
					},
				}},
				Status: "HTTP/1.1 200 OK",
			}},
		}},
		wantErr: errInvalidResponse,
		// default of http.responseWriter
		wantCode: http.StatusOK,
	}, {
		desc: "bad: multiple hrefs and no status",
		responses: []response{{
			Href: []string{"http://example.com/foo", "http://example.com/bar"},
		}},
		wantErr: errInvalidResponse,
		// default of http.responseWriter
		wantCode: http.StatusOK,
	}, {
		desc: "bad: one href and no propstat",
		responses: []response{{
			Href: []string{"http://example.com/foo"},
		}},
		wantErr: errInvalidResponse,
		// default of http.responseWriter
		wantCode: http.StatusOK,
	}, {
		desc: "bad: status with one href and propstat",
		responses: []response{{
			Href: []string{"http://example.com/foo"},
			Propstat: []propstat{{
				Prop: []Property{{
					XMLName: xml.Name{
						Space: "http://example.com/",
						Local: "foo",
					},
				}},
				Status: "HTTP/1.1 200 OK",
			}},
			Status: "HTTP/1.1 200 OK",
		}},
		wantErr: errInvalidResponse,
		// default of http.responseWriter
		wantCode: http.StatusOK,
	}, {
		desc: "bad: multiple hrefs and propstat",
		responses: []response{{
			Href: []string{
				"http://example.com/foo",
				"http://example.com/bar",
			},
			Propstat: []propstat{{
				Prop: []Property{{
					XMLName: xml.Name{
						Space: "http://example.com/",
						Local: "foo",
					},
				}},
				Status: "HTTP/1.1 200 OK",
			}},
		}},
		wantErr: errInvalidResponse,
		// default of http.responseWriter
		wantCode: http.StatusOK,
	}}

loop:
	for _, tc := range testCases {
		rec := httptest.NewRecorder()
		w := multistatusWriter{w: rec, responseDescription: tc.respdesc}
		if tc.writeHeader {
			if err := w.writeHeader(); err != nil {
				t.Errorf("%s: got writeHeader error %v, want nil", tc.desc, err)
				continue
			}
		}
		for _, r := range tc.responses {
			if err := w.write(&r); err != nil {
				if err != tc.wantErr {
					t.Errorf("%s: got write error %v, want %v",
						tc.desc, err, tc.wantErr)
				}
				continue loop
			}
		}
		if err := w.close(); err != tc.wantErr {
			t.Errorf("%s: got close error %v, want %v",
				tc.desc, err, tc.wantErr)
			continue
		}
		if rec.Code != tc.wantCode {
			t.Errorf("%s: got HTTP status code %d, want %d\n",
				tc.desc, rec.Code, tc.wantCode)
			continue
		}

		// normalize returns the normalized XML content of s. In contrast to
		// the WebDAV specification, it ignores whitespace within property
		// values of mixed XML content.
		normalize := func(s string) string {
			d := xml.NewDecoder(strings.NewReader(s))
			var b bytes.Buffer
			e := xml.NewEncoder(&b)
			for {
				tok, err := d.Token()
				if err != nil {
					if err == io.EOF {
						break
					}
					t.Fatalf("%s: Token %v", tc.desc, err)
				}
				switch val := tok.(type) {
				case xml.Comment, xml.Directive, xml.ProcInst:
					continue
				case xml.CharData:
					if len(bytes.TrimSpace(val)) == 0 {
						continue
					}
				}
				if err := e.EncodeToken(tok); err != nil {
					t.Fatalf("%s: EncodeToken: %v", tc.desc, err)
				}
			}
			if err := e.Flush(); err != nil {
				t.Fatalf("%s: Flush: %v", tc.desc, err)
			}
			return b.String()
		}

		gotXML := normalize(rec.Body.String())
		wantXML := normalize(tc.wantXML)
		if gotXML != wantXML {
			t.Errorf("%s: XML body\ngot  %q\nwant %q", tc.desc, gotXML, wantXML)
		}
	}
}

func TestReadProppatch(t *testing.T) {
	// TODO(rost): These "golden XML" tests easily break with changes in the
	// xml package. A whitespace-preserving normalizer of XML content is
	// required to make these tests more robust.
	testCases := []struct {
		desc       string
		input      string
		wantPP     []Proppatch
		wantStatus int
	}{{
		desc: "proppatch: section 9.2",
		input: `` +
			`<?xml version="1.0" encoding="utf-8" ?>` +
			`<D:propertyupdate xmlns:D="DAV:"` +
			`                  xmlns:Z="http://ns.example.com/z/">` +
			`    <D:set>` +
			`         <D:prop>` +
			`              <Z:Authors>` +
			`              <Z:Author>Jim Whitehead</Z:Author>` +
			`              <Z:Author>Roy Fielding</Z:Author>` +
			`              </Z:Authors>` +
			`         </D:prop>` +
			`    </D:set>` +
			`    <D:remove>` +
			`         <D:prop><Z:Copyright-Owner/></D:prop>` +
			`    </D:remove>` +
			`</D:propertyupdate>`,
		wantPP: []Proppatch{{
			Props: []Property{{
				xml.Name{Space: "http://ns.example.com/z/", Local: "Authors"},
				"",
				[]byte(`` +
					`              ` +
					`<z:Author xmlns:z="http://ns.example.com/z/">` +
					`Jim Whitehead` +
					`</z:Author>` +
					`              ` +
					`<z:Author xmlns:z="http://ns.example.com/z/">` +
					`Roy Fielding` +
					`</z:Author>` +
					`              `,
				),
			}},
		}, {
			Remove: true,
			Props: []Property{{
				xml.Name{Space: "http://ns.example.com/z/", Local: "Copyright-Owner"},
				"",
				nil,
			}},
		}},
	}, {
		desc: "proppatch: section 4.3.1 (mixed content)",
		input: `` +
			`<?xml version="1.0" encoding="utf-8" ?>` +
			`<D:propertyupdate xmlns:D="DAV:"` +
			`                  xmlns:Z="http://ns.example.com/z/">` +
			`  <D:set>` +
			`     <D:prop xml:lang="en" xmlns:D="DAV:">` +
			`         <x:author xmlns:x='http://example.com/ns'>` +
			`            <x:name>Jane Doe</x:name>` +
			`            <!-- Jane's contact info -->` +
			`            <x:uri type='email'` +
			`                   added='2005-11-26'>mailto:jane.doe@example.com</x:uri>` +
			`            <x:uri type='web'` +
			`                   added='2005-11-27'>http://www.example.com</x:uri>` +
			`            <x:notes xmlns:h='http://www.w3.org/1999/xhtml'>` +
			`              Jane has been working way <h:em>too</h:em> long on the` +
			`              long-awaited revision of <![CDATA[<RFC2518>]]>.` +
			`            </x:notes>` +
			`         </x:author>` +
			`     </D:prop>` +
			`  </D:set>` +
			`</D:propertyupdate>`,
		wantPP: []Proppatch{{
			Props: []Property{{
				xml.Name{Space: "http://example.com/ns", Local: "author"},
				"en",
				[]byte(`` +
					`            ` +
					`<ns:name xmlns:ns="http://example.com/ns">Jane Doe</ns:name>` +
					`                        ` +
					`<ns:uri xmlns:ns="http://example.com/ns" type="email" added="2005-11-26">` +
					`mailto:jane.doe@example.com` +
					`</ns:uri>` +
					`            ` +
					`<ns:uri xmlns:ns="http://example.com/ns" type="web" added="2005-11-27">` +
					`http://www.example.com` +
					`</ns:uri>` +
					`            ` +

					`<ns:notes xmlns:ns="http://example.com/ns"` +
					` xmlns:h="http://www.w3.org/1999/xhtml">` +
					`             ` +
					` Jane has been working way` +
					` <h:em>too</h:em>` +
					` long on the` + `             ` +
					` long-awaited revision of &lt;RFC2518&gt;.` +
					`            ` +
					`</ns:notes>` +
					`         `,
				),
			}},
		}},
	}, {
		desc: "proppatch: lang attribute on prop",
		input: `` +
			`<?xml version="1.0" encoding="utf-8" ?>` +
			`<D:propertyupdate xmlns:D="DAV:">` +
			`    <D:set>` +
			`         <D:prop xml:lang="en">` +
			`              <foo xmlns="http://example.com/ns"/>` +
			`         </D:prop>` +
			`    </D:set>` +
			`</D:propertyupdate>`,
		wantPP: []Proppatch{{
			Props: []Property{{
				xml.Name{Space: "http://example.com/ns", Local: "foo"},
				"en",
				nil,
			}},
		}},
	}, {
		desc: "bad: remove with value",
		input: `` +
			`<?xml version="1.0" encoding="utf-8" ?>` +
			`<D:propertyupdate xmlns:D="DAV:"` +
			`                  xmlns:Z="http://ns.example.com/z/">` +
			`    <D:remove>` +
			`         <D:prop>` +
			`              <Z:Authors>` +
			`              <Z:Author>Jim Whitehead</Z:Author>` +
			`              </Z:Authors>` +
			`         </D:prop>` +
			`    </D:remove>` +
			`</D:propertyupdate>`,
		wantStatus: http.StatusBadRequest,
	}, {
		desc: "bad: empty propertyupdate",
		input: `` +
			`<?xml version="1.0" encoding="utf-8" ?>` +
			`<D:propertyupdate xmlns:D="DAV:"` +
			`</D:propertyupdate>`,
		wantStatus: http.StatusBadRequest,
	}, {
		desc: "bad: empty prop",
		input: `` +
			`<?xml version="1.0" encoding="utf-8" ?>` +
			`<D:propertyupdate xmlns:D="DAV:"` +
			`                  xmlns:Z="http://ns.example.com/z/">` +
			`    <D:remove>` +
			`        <D:prop/>` +
			`    </D:remove>` +
			`</D:propertyupdate>`,
		wantStatus: http.StatusBadRequest,
	}}

	for _, tc := range testCases {
		pp, status, err := readProppatch(strings.NewReader(tc.input))
		if tc.wantStatus != 0 {
			if err == nil {
				t.Errorf("%s: got nil error, want non-nil", tc.desc)
				continue
			}
		} else if err != nil {
			t.Errorf("%s: %v", tc.desc, err)
			continue
		}
		if status != tc.wantStatus {
			t.Errorf("%s: got status %d, want %d", tc.desc, status, tc.wantStatus)
			continue
		}
		if !reflect.DeepEqual(pp, tc.wantPP) || status != tc.wantStatus {
			t.Errorf("%s: proppatch\ngot  %v\nwant %v", tc.desc, pp, tc.wantPP)
		}
	}
}
