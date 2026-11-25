// HTML `pattern` attributes are compiled with the RegExp `v` flag, where a
// literal "-" inside [...] must be escaped. Keep in sync with internal/api.
export const HOST_PATTERN = String.raw`[a-z0-9]([a-z0-9\-]*[a-z0-9])?`;
export const USER_PATTERN = String.raw`[a-z][a-z0-9_\-]+`;
