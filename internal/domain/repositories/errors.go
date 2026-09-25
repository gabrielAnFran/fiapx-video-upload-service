package repositories

import "errors"

// ErrNotFound is returned by repository lookups that find nothing.
var ErrNotFound = errors.New("not found")

// ErrDuplicateEmail is returned when registering a user whose email is
// already taken.
var ErrDuplicateEmail = errors.New("email already registered")

// ErrForbidden is returned when a user attempts to access a video they do
// not own.
var ErrForbidden = errors.New("forbidden")

// ErrNotReady is returned when a download URL is requested for a video that
// has not finished processing yet.
var ErrNotReady = errors.New("video not ready for download")
