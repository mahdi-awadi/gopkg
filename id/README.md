# id

Identifier generators (currently UUIDv7). Stdlib-only.

```
go get github.com/mahdi-awadi/gopkg/id@latest
```

## Why UUIDv7

UUIDv7 (RFC 9562) is time-ordered: the top 48 bits are a millisecond
Unix timestamp. That keeps B-tree indexes dense — unlike UUIDv4 which
inserts everywhere in the index and fragments it.

## Usage

```go
import "github.com/mahdi-awadi/gopkg/id"

u := id.UUIDv7()        // "018f4b5c-a29b-7aa1-b3cd-9e4a22e8f100"
raw := id.UUIDv7Raw()   // [16]byte — for binary UUID columns
```

## API

- `UUIDv7() string` — hyphenated 36-char RFC 9562 v7 UUID
- `UUIDv7Raw() [16]byte` — raw bytes

Both are safe for concurrent use.

## License

[MIT](../LICENSE)
