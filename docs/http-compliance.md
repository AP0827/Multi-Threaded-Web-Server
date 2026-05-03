# MTWS HTTP/1.1 Compliance Profile

MTWS intentionally implements a strict HTTP/1.1 subset for research purposes.
The goal is not to be a general-purpose web server; the goal is to eliminate
security ambiguity by accepting only request forms that the parser and in-parser
WAF interpret unambiguously.

## Supported

- HTTP version: `HTTP/1.1`
- Request target: origin-form paths beginning with `/`
- Request target special case: `OPTIONS *`
- Headers: canonicalized field names with RFC token syntax
- Body framing: `Content-Length`
- Body framing: `Transfer-Encoding: chunked`
- Chunk extensions: accepted and ignored after size parsing
- Trailers: accepted with strict limits and WAF scanning
- WAF scan fields: URI, header values, decoded body bytes, trailer values
- Response connection behavior: HTTP/1.1 keep-alive by default
- Response connection behavior: `Connection: close` when requested or when max
  keep-alive requests is reached
- Limits:
- Request line: 4096 bytes
- Header line: 8192 bytes
- Header count: 64
- Trailer count: 16
- Body size: 1 MiB

## Rejected

- HTTP versions other than `HTTP/1.1`
- Missing or empty `Host`
- `Host` values containing whitespace or control characters
- Duplicate headers
- Obsolete folded headers
- Invalid header field names
- Absolute-form targets, authority-form targets, and non-`OPTIONS` asterisk targets
- Any request target containing control characters
- Requests containing both `Content-Length` and `Transfer-Encoding`
- Transfer codings other than exactly `chunked`
- Invalid chunk sizes
- Invalid chunk terminators
- Unlimited persistent HTTP/1.1 connection reuse
- Forbidden trailer fields:
- `Content-Length`
- `Host`
- `Transfer-Encoding`
- `Trailer`

## Research Rationale

Rejecting ambiguous inputs is part of the security model. MTWS should fail
closed whenever a request requires a second interpretation step, compatibility
guess, or downstream parser behavior to decide what the request means.

This makes MTWS narrower than a production HTTP server, but it makes the
parser/WAF relationship clearer: the same parser that accepts the request is the
parser whose interpretation was inspected.
