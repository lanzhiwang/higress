package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/higress-group/wasm-go/pkg/tokenusage"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/stretchr/testify/require"
)

// stubCtx 只实现 tokenusage 实际用到的 ctx 方法, 其余方法由嵌入的接口满足类型.
type stubCtx struct {
	wrapper.HttpContext
	ctx   map[string]interface{}
	attrs map[string]interface{}
}

func newStubCtx() *stubCtx {
	return &stubCtx{ctx: map[string]interface{}{}, attrs: map[string]interface{}{}}
}

func (c *stubCtx) SetContext(k string, v interface{})       { c.ctx[k] = v }
func (c *stubCtx) GetContext(k string) interface{}          { return c.ctx[k] }
func (c *stubCtx) SetUserAttribute(k string, v interface{}) { c.attrs[k] = v }
func (c *stubCtx) GetUserAttribute(k string) interface{}    { return c.attrs[k] }

func (c *stubCtx) GetBoolContext(k string, d bool) bool {
	if v, ok := c.ctx[k].(bool); ok {
		return v
	}
	return d
}

func (c *stubCtx) GetStringContext(k, d string) string {
	if v, ok := c.ctx[k].(string); ok {
		return v
	}
	return d
}

func (c *stubCtx) GetByteSliceContext(k string, d []byte) []byte {
	if v, ok := c.ctx[k].([]byte); ok {
		return v
	}
	return d
}

// typeLastCompletedEvent 复刻 Vendor3/DeepSeek-V4.1-Flash 风格的 response.completed 事件:
// JSON 中 "usage" 出现在 "type":"response.completed" 之前 (type 在事件末尾).
// 这种字段顺序会让 tokenusage 内部的 mergeLargeResponseAPIChunks 启发式永不触发,
// 因此该事件一旦被回调边界切开, usage 就无法恢复.
func typeLastCompletedEvent() []byte {
	answer := strings.Repeat("这是一段回答文本。", 60)
	return []byte("event: response.completed\n" +
		`data: {"response":{"created_at":1790211447,"id":"cds_test","model":"Vendor3/DeepSeek-V4.1-Flash",` +
		`"object":"response","output":[{"content":[{"text":"` + answer + `","type":"output_text"}],` +
		`"id":"msg_test","role":"assistant","status":"completed","type":"message"}],` +
		`"reasoning":{"effort":"medium"},"service_tier":"default","status":"completed","store":true,` +
		`"usage":{"input_tokens":59,"input_tokens_details":{"cached_tokens":0},"output_tokens":419,` +
		`"output_tokens_details":{"reasoning_tokens":154},"total_tokens":478}},` +
		`"sequence_number":113,"type":"response.completed"}` + "\n\n")
}

// TestUsageRecoveredWhenCompletedEventIsSplit 断言: 无论回调边界落在事件内部的哪个
// 字节位置, 分帧后 usage 都能被完整解析出来.
func TestUsageRecoveredWhenCompletedEventIsSplit(t *testing.T) {
	event := typeLastCompletedEvent()

	// 先确认 fixture 确实复刻了问题顺序, 否则这个测试就失去意义.
	require.Less(t, bytes.Index(event, []byte(`"usage"`)), bytes.Index(event, []byte(`"response.completed"`)),
		"fixture 必须复刻 usage 在 type 之前的字段顺序")

	for cut := 1; cut < len(event); cut++ {
		ctx := newStubCtx()
		framer := &sseFramer{}
		var total int64
		for _, chunk := range [][]byte{event[:cut], event[cut:]} {
			for _, ev := range framer.frameCallback(chunk) {
				if u := tokenusage.GetTokenUsage(ctx, ev); u.TotalToken > 0 {
					total = u.TotalToken
				}
			}
		}
		framer.drain()
		require.Equalf(t, int64(478), total, "在字节偏移 %d 处切开后 usage 丢失", cut)
	}
}

// TestUsageRecoveredWhenEventSplitAcrossManyCallbacks 断言逐字节投递 (最坏切分) 也能恢复.
func TestUsageRecoveredWhenEventSplitAcrossManyCallbacks(t *testing.T) {
	event := typeLastCompletedEvent()
	ctx := newStubCtx()
	framer := &sseFramer{}
	var total int64
	for i := 0; i < len(event); i++ {
		for _, ev := range framer.frameCallback(event[i : i+1]) {
			if u := tokenusage.GetTokenUsage(ctx, ev); u.TotalToken > 0 {
				total = u.TotalToken
			}
		}
	}
	framer.drain()
	require.Equal(t, int64(478), total)
}

// TestUnterminatedTailIsDropped 断言流结束时未以分隔符结尾的残留会被丢弃且不产生计费值.
func TestUnterminatedTailIsDropped(t *testing.T) {
	event := typeLastCompletedEvent()
	ctx := newStubCtx()
	framer := &sseFramer{}
	var total int64
	for _, ev := range framer.frameCallback(bytes.TrimRight(event, "\n")) {
		if u := tokenusage.GetTokenUsage(ctx, ev); u.TotalToken > 0 {
			total = u.TotalToken
		}
	}
	require.Equal(t, int64(0), total)
	require.Zero(t, framer.drain())
}
