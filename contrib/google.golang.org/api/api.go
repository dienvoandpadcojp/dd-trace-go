// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

// Package api provides functions to trace the google.golang.org/api package.
//
// WARNING: Please note we periodically re-generate the endpoint metadata that is used to enrich some tags
// added by this integration using the latest versions of github.com/googleapis/google-api-go-client (which does not
// follow semver due to the auto-generated nature of the package). For this reason, there might be unexpected changes
// in some tag values like service.name and resource.name, depending on the google.golang.org/api that you are using in your
// project. If this is not an acceptable behavior for your use-case, you can disable this feature using the
// WithEndpointMetadataDisabled option.
package api // import "gopkg.in/DataDog/dd-trace-go.v1/contrib/google.golang.org/api"

import (
	"math"
	"net/http"

	"gopkg.in/DataDog/dd-trace-go.v1/contrib/google.golang.org/api/internal"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/telemetry"

	"golang.org/x/oauth2/google"
)

const componentName = "google.golang.org/api"

func init() {
	telemetry.LoadIntegration(componentName)
}

// apiEndpoints are all of the defined endpoints for the Google API; it is populated
// by "go generate".
var apiEndpoints *internal.Tree

// NewClient creates a new oauth http client suitable for use with the google
// APIs with all requests traced automatically.
func NewClient(options ...Option) (*http.Client, error) {
	cfg := newConfig(options...)
	log.Debug("contrib/google.golang.org/api: Creating Client: %#v", cfg)
	client, err := google.DefaultClient(cfg.ctx, cfg.scopes...)
	if err != nil {
		return nil, err
	}
	client.Transport = WrapRoundTripper(client.Transport, options...)
	return client, nil
}

// WrapRoundTripper wraps a RoundTripper intended for interfacing with
// Google APIs and traces all requests.
func WrapRoundTripper(transport http.RoundTripper, options ...Option) http.RoundTripper {
	cfg := newConfig(options...)
	log.Debug("contrib/google.golang.org/api: Wrapping RoundTripper: %#v", cfg)
	rtOpts := []httptrace.RoundTripperOption{
		httptrace.WithBefore(func(req *http.Request, span ddtrace.Span) {
			if !cfg.endpointMetadataDisabled {
				setTagsWithEndpointMetadata(req, span)
			} else {
				setTagsWithoutEndpointMetadata(req, span)
			}
			if cfg.serviceName != "" {
				span.SetTag(ext.ServiceName, cfg.serviceName)
			}
			span.SetTag(ext.Component, componentName)
			span.SetTag(ext.SpanKind, ext.SpanKindClient)
		}),
	}
	if !math.IsNaN(cfg.analyticsRate) {
		rtOpts = append(rtOpts, httptrace.RTWithAnalyticsRate(cfg.analyticsRate))
	}
	return httptrace.WrapRoundTripper(transport, rtOpts...)
}

func setTagsWithEndpointMetadata(req *http.Request, span ddtrace.Span) {
	e, ok := apiEndpoints.Get(req.URL.Hostname(), req.Method, req.URL.Path)
	if ok {
		span.SetTag(ext.ServiceName, e.ServiceName)
		span.SetTag(ext.ResourceName, e.ResourceName)
	} else {
		setTagsWithoutEndpointMetadata(req, span)
	}
}

func setTagsWithoutEndpointMetadata(req *http.Request, span ddtrace.Span) {
	span.SetTag(ext.ServiceName, "google")
	span.SetTag(ext.ResourceName, req.Method+" "+req.URL.Hostname())
}
