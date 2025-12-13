// https://chromium.googlesource.com/v8/v8/+/refs/heads/main/test/mjsunit/date-parse.js

package rss

import (
	"testing"
	"time"
)

// Test all the formats in UT timezone.
var testCasesUT = []string{"Sat, 01-Jan-2000 08:00:00 UT",
	"Sat, 01 Jan 2000 08:00:00 UT",
	"Jan 01 2000 08:00:00 UT",
	"Jan 01 08:00:00 UT 2000",
	"Saturday, 01-Jan-00 08:00:00 UT",
	"01 Jan 00 08:00 +0000",
	// Ignore weekdays.
	"Mon, 01 Jan 2000 08:00:00 UT",
	"Tue, 01 Jan 2000 08:00:00 UT",
	// Ignore prefix that is not part of a date.
	"[Saturday] Jan 01 08:00:00 UT 2000",
	"Ignore all of this stuff because it is annoying 01 Jan 2000 08:00:00 UT",
	"[Saturday] Jan 01 2000 08:00:00 UT",
	"All of this stuff is really annoying, so it will be ignored Jan 01 2000 08:00:00 UT",
	// If the three first letters of the month is a
	// month name we are happy - ignore the rest.
	"Sat, 01-Janisamonth-2000 08:00:00 UT",
	"Sat, 01 Janisamonth 2000 08:00:00 UT",
	"Janisamonth 01 2000 08:00:00 UT",
	"Janisamonth 01 08:00:00 UT 2000",
	"Saturday, 01-Janisamonth-00 08:00:00 UT",
	"01 Janisamonth 00 08:00 +0000",
	// Allow missing space between month and day.
	"Janisamonthandtherestisignored01 2000 08:00:00 UT",
	"Jan01 2000 08:00:00 UT",
	// Allow year/month/day format.
	"Sat, 2000/01/01 08:00:00 UT",
	// Allow month/day/year format.
	"Sat, 01/01/2000 08:00:00 UT",
	// Allow month/day year format.
	"Sat, 01/01 2000 08:00:00 UT",
	// Allow comma instead of space after day, month and year.
	"Sat, 01,Jan,2000,08:00:00 UT",
	// Seconds are optional.
	"Sat, 01-Jan-2000 08:00 UT",
	"Sat, 01 Jan 2000 08:00 UT",
	"Jan 01 2000 08:00 UT",
	"Jan 01 08:00 UT 2000",
	"Saturday, 01-Jan-00 08:00 UT",
	"01 Jan 00 08:00 +0000",
	// Allow AM/PM after the time.
	"Sat, 01-Jan-2000 08:00 AM UT",
	"Sat, 01 Jan 2000 08:00 AM UT",
	"Jan 01 2000 08:00 AM UT",
	"Jan 01 08:00 AM UT 2000",
	"Saturday, 01-Jan-00 08:00 AM UT",
	"01 Jan 00 08:00 AM +0000",
	// White space and stuff in parenthesis is
	// apparently allowed in most places where white
	// space is allowed.
	"   Sat,   01-Jan-2000   08:00:00   UT  ",
	"  Sat,   01   Jan   2000   08:00:00   UT  ",
	"  Saturday,   01-Jan-00   08:00:00   UT  ",
	"  01    Jan   00    08:00   +0000   ",
	" ()(Sat, 01-Jan-2000)  Sat,   01-Jan-2000   08:00:00   UT  ",
	"  Sat()(Sat, 01-Jan-2000)01   Jan   2000   08:00:00   UT  ",
	"  Sat,(02)01   Jan   2000   08:00:00   UT  ",
	"  Sat,  01(02)Jan   2000   08:00:00   UT  ",
	"  Sat,  01  Jan  2000 (2001)08:00:00   UT  ",
	"  Sat,  01  Jan  2000 (01)08:00:00   UT  ",
	"  Sat,  01  Jan  2000 (01:00:00)08:00:00   UT  ",
	"  Sat,  01  Jan  2000  08:00:00 (CDT)UT  ",
	"  Sat,  01  Jan  2000  08:00:00  UT((((CDT))))",
	"  Saturday,   01-Jan-00 ()(((asfd)))(Sat, 01-Jan-2000)08:00:00   UT  ",
	"  01    Jan   00    08:00 ()(((asdf)))(Sat, 01-Jan-2000)+0000   ",
	"  01    Jan   00    08:00   +0000()((asfd)(Sat, 01-Jan-2000)) "}

// Test that we do the right correction for different time zones.
// I"ll assume that we can handle the same formats as for UT and only
// test a few formats for each of the timezones.
//
// GMT = UT
var testCasesGMT = []string{
	"Sat, 01-Jan-2000 08:00:00 GMT",
	"Sat, 01-Jan-2000 08:00:00 GMT+0",
	"Sat, 01-Jan-2000 08:00:00 GMT+00",
	"Sat, 01-Jan-2000 08:00:00 GMT+000",
	"Sat, 01-Jan-2000 08:00:00 GMT+0000",
	"Sat, 01-Jan-2000 08:00:00 GMT+00:00", // Interestingly, KJS cannot handle this.
	"Sat, 01 Jan 2000 08:00:00 GMT",
	"Saturday, 01-Jan-00 08:00:00 GMT",
	"01 Jan 00 08:00 -0000",
	"01 Jan 00 08:00 +0000"}

// EST = UT minus 5 hours.
var testCasesEST = []string{
	"Sat, 01-Jan-2000 03:00:00 UTC-0500",
	"Sat, 01-Jan-2000 03:00:00 UTC-05:00", // Interestingly, KJS cannot handle this.
	"Sat, 01-Jan-2000 03:00:00 EST",
	"Sat, 01 Jan 2000 03:00:00 EST",
	"Saturday, 01-Jan-00 03:00:00 EST",
	"01 Jan 00 03:00 -0500"}

// EDT = UT minus 4 hours.
var testCasesEDT = []string{
	"Sat, 01-Jan-2000 04:00:00 EDT",
	"Sat, 01 Jan 2000 04:00:00 EDT",
	"Saturday, 01-Jan-00 04:00:00 EDT",
	"01 Jan 00 04:00 -0400"}

// CST = UT minus 6 hours.
var testCasesCST = []string{
	"Sat, 01-Jan-2000 02:00:00 CST",
	"Sat, 01 Jan 2000 02:00:00 CST",
	"Saturday, 01-Jan-00 02:00:00 CST",
	"01 Jan 00 02:00 -0600"}

// CDT = UT minus 5 hours.
var testCasesCDT = []string{
	"Sat, 01-Jan-2000 03:00:00 CDT",
	"Sat, 01 Jan 2000 03:00:00 CDT",
	"Saturday, 01-Jan-00 03:00:00 CDT",
	"01 Jan 00 03:00 -0500"}

// MST = UT minus 7 hours.
var testCasesMST = []string{
	"Sat, 01-Jan-2000 01:00:00 MST",
	"Sat, 01 Jan 2000 01:00:00 MST",
	"Saturday, 01-Jan-00 01:00:00 MST",
	"01 Jan 00 01:00 -0700"}

// MDT = UT minus 6 hours.
var testCasesMDT = []string{
	"Sat, 01-Jan-2000 02:00:00 MDT",
	"Sat, 01 Jan 2000 02:00:00 MDT",
	"Saturday, 01-Jan-00 02:00:00 MDT",
	"01 Jan 00 02:00 -0600"}

// PST = UT minus 8 hours.
var testCasesPST = []string{
	"Sat, 01-Jan-2000 00:00:00 PST",
	"Sat, 01 Jan 2000 00:00:00 PST",
	"Saturday, 01-Jan-00 00:00:00 PST",
	"01 Jan 00 00:00 -0800",
	// Allow missing time.
	"Sat, 01-Jan-2000 PST"}

// PDT = UT minus 7 hours.
var testCasesPDT = []string{
	"Sat, 01-Jan-2000 01:00:00 PDT",
	"Sat, 01 Jan 2000 01:00:00 PDT",
	"Saturday, 01-Jan-00 01:00:00 PDT",
	"01 Jan 00 01:00 -0700"}

// Local time cases.
var testCasesLocalTime = []string{
	// Allow timezone omission.
	"Sat, 01-Jan-2000 08:00:00",
	"Sat, 01 Jan 2000 08:00:00",
	"Jan 01 2000 08:00:00",
	"Jan 01 08:00:00 2000",
	"Saturday, 01-Jan-00 08:00:00",
	"01 Jan 00 08:00"}

type testCaseMisc struct {
	date string
	unix int64
}

// Misc. test cases that result in a different time value.
var testCasesMisc = []testCaseMisc{
	// Special handling for years in the [0, 100) range.
	{"Sat, 01 Jan 0 08:00:00 UT", 946713600000},      // year 2000
	{"Sat, 01 Jan 49 08:00:00 UT", 2493100800000},    // year 2049
	{"Sat, 01 Jan 50 08:00:00 UT", -631123200000},    // year 1950
	{"Sat, 01 Jan 99 08:00:00 UT", 915177600000},     // year 1999
	{"Sat, 01 Jan 100 08:00:00 UT", -59011430400000}, // year 100
	// Test PM after time.
	{"Sat, 01-Jan-2000 08:00 PM UT", 946756800000},
	{"Sat, 01 Jan 2000 08:00 PM UT", 946756800000},
	{"Jan 01 2000 08:00 PM UT", 946756800000},
	{"Jan 01 08:00 PM UT 2000", 946756800000},
	{"Saturday, 01-Jan-00 08:00 PM UT", 946756800000},
	{"01 Jan 00 08:00 PM +0000", 946756800000}}

// Test different version of the ES5 date time string format.
var testCasesES5Misc = []testCaseMisc{
	{"2000-01-01T08:00:00.000Z", 946713600000},
	{"2000-01-01T08:00:00Z", 946713600000},
	{"2000-01-01T08:00Z", 946713600000},
	{"2000-01T08:00:00.000Z", 946713600000},
	{"2000T08:00:00.000Z", 946713600000},
	{"2000T08:00Z", 946713600000},
	{"2000-01T00:00:00.000-08:00", 946713600000},
	{"2000-01T08:00:00.001Z", 946713600001},
	{"2000-01T08:00:00.099Z", 946713600099},
	{"2000-01T08:00:00.999Z", 946713600999},
	{"2000-01T00:00:00.001-08:00", 946713600001},
	{"2000-01-01T24:00Z", 946771200000},
	{"2000-01-01T24:00:00Z", 946771200000},
	{"2000-01-01T24:00:00.000Z", 946771200000},
	{"2000-01-01T24:00:00.000Z", 946771200000}}

var testCasesES5MiscNegative = []string{
	"2000-01-01TZ",
	"2000-01-01T60Z",
	"2000-01-01T60:60Z",
	"2000-01-0108:00Z",
	"2000-01-01T08Z",
	"2000-01-01T24:01",
	"2000-01-01T24:00:01",
	"2000-01-01T24:00:00.001",
	"2000-01-01T24:00:00.999Z"}

var _, offset = time.Now().Zone()
var localOffset = -int64(offset) * 1000

var testCasesES2016TZ = []testCaseMisc{
	// If the timezone is absent and time is present, use local time
	{"2000-01-02T00:00", 946771200000 + localOffset},
	{"2000-01-02T00:00:00", 946771200000 + localOffset},
	{"2000-01-02T00:00:00.000", 946771200000 + localOffset},
	// If timezone is absent and time is absent, use UTC
	{"2000-01-02", 946771200000},
	{"2000-01-02", 946771200000},
	{"2000-01-02", 946771200000},
}

// Negative tests.
var testCasesNegative = []string{
	"May 25 2008 1:30 (PM)) UTC", // Bad unmatched ')' after number.
	"May 25 2008 1:30( )AM (PM)", //
	"a1",                         // Issue 126448, 53209.
	"nasfdjklsfjoaifg1",
	"x_2",
	"May 25 2008 AAA (GMT)"} // Unknown word after number.

func TestDateParse(t *testing.T) {
	for _, testCases := range [][]string{
		testCasesUT,
		testCasesGMT,
		testCasesEST,
		testCasesEDT,
		testCasesCST,
		testCasesCDT,
		testCasesMST,
		testCasesMDT,
		testCasesPST,
		testCasesPDT,
	} {
		for _, testCase := range testCases {
			date := parseDate(testCase)
			if date == nil {
				t.Fatalf("fail to parse %#v", testCase)
			}
			const want = 946713600000
			have := date.UnixMilli()
			if have != want {
				t.Fatalf("parse %#v: want %d, have %d", testCase, want, have)
			}
		}
	}
}

func TestDateParseLocalTime(t *testing.T) {
	for _, testCase := range testCasesLocalTime {
		date := parseDate(testCase)
		if date == nil || date.UnixMilli() == 0 {
			t.Fatalf("fail to parse %#v", testCase)
		}
	}
}

func TestDateParseMisc(t *testing.T) {
	for _, testCases := range [][]testCaseMisc{testCasesMisc, testCasesES5Misc, testCasesES2016TZ} {
		for _, testCase := range testCases {
			date := parseDate(testCase.date)
			if date == nil {
				t.Fatalf("fail to parse %#v", testCase.date)
			}
			have := date.UnixMilli()
			if have != testCase.unix {
				t.Fatalf("parse %#v: want %d, have %d", testCase.date, testCase.unix, have)
			}
		}
	}
}

// Dates from 1970 to ~2070 with 150h steps.
func TestDateParseRFC3339(t *testing.T) {
	for i := int64(0); i < 24*365*100; i += 150 {
		testCase := time.Unix(i*3600, 0)
		dateString := testCase.Format(time.RFC3339)
		date := parseDate(dateString)
		if date == nil {
			t.Fatalf("fail to parse %#v", dateString)
		}
		want := testCase.UnixMilli()
		have := date.UnixMilli()
		if have != want {
			t.Fatalf("parse %#v: want %d, have %d", dateString, want, have)
		}
	}
}

func TestDateParseNegative(t *testing.T) {
	for _, testCases := range [][]string{testCasesES5MiscNegative, testCasesNegative} {
		for _, testCase := range testCases {
			if parseDate(testCase) != nil {
				t.Fatalf("successfully parsed %#v", testCase)
			}
		}
	}
}
