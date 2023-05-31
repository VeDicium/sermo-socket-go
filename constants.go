package sermo

import "regexp"

var PARAM_REGEX = regexp.MustCompile("(?::([A-Za-z0-9_]+))")
