// SPDX-License-Identifier: MIT OR LGPL-3.0-or-later

// Package testacme provides a testing focused helper library which enables test
// authors to execute ACME integration and functional tests. The ACME server
// exercises challenge verifications with support for local-only environments by
// way of an included DNS resolver and convenience wrappers for Pebble.
//
// https://github.com/letsencrypt/pebble
//
// See package's test source files for example usages.
package testacme
