# Tomori reference

Sento's AI transport and UI lifecycle follow the OpenAI-compatible generation,
queued UI updates, cancellation, and generation-counter patterns in
`tomori/ai.go` at `ce4db21ed717c35dc044ca9b5312cfdc118274ea`.

Tomori is MIT licensed, copyright 2026 Eric Julianto (same copyright holder
and MIT terms as Sento's root LICENSE). Sento uses its own request-generation
prompt and validates generated output against its .http parser.
