# Release Notes

## Version 2.1.0

### Key Changes
- **XML Support**: Full support for XML encoding of arbitrary models, maps, slices, and primitive types.
- **Extensible Response Formats**: Introduced the `ResponseEncoder` registry system allowing third-party packages to plug in custom serialization formats (such as SOAP).
- **Auto-Content Negotiation**: Implemented HTTP `Accept` header content negotiation when format is set to `"both"`.
- **APIHandler Wrapper**: Added `APIHandler` interface type and `Handler` / `HandlerWithFormat` adapter wrappers, removing `return` statement boilerplate from HTTP handlers.
- **Proper Error Encoding**: Redesigned serialization fallback to correctly marshal nested `error` interfaces as strings.
- **Logger v2 Upgrade**: Upgraded core logger dependency to use `github.com/danmaina/logger/v2`.
