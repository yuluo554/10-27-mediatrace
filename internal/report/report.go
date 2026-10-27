// Package report 定义三态判定结果与证据四元组，并做**渲染前校验**。
//
// 纪律（plan/03 §一、data/README §5.7）：
//
//	· 每条 fail 必须带 byte_offset + field_path + values + delta，缺一即拒绝输出；
//	· gray 只能来自 data/README §5.5 的封闭枚举，不得兜住解析错误；
//	· 判定与呈现分家：规则层只产 Finding，渲染层只消费 report.Report。
package report

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Status 是三态判定结果。
type Status string

const (
	Pass Status = "pass"
	Fail Status = "fail"
	Gray Status = "gray"
)

// GrayReason 是 gray 的封闭枚举（data/README.md §5.5）。
// 新增原因必须先改 data/README §5.5 与 docs/report-schema.md。
type GrayReason string

const (
	GrayVFRDetected         GrayReason = "vfr_detected"
	GrayNoRollDistance      GrayReason = "no_roll_distance"
	GrayUnknownCodecPriming GrayReason = "unknown_codec_priming"
	GrayMediaRateNotOne     GrayReason = "media_rate_not_one"
	GrayNoEdts              GrayReason = "no_edts"
	GrayHDRMetadataMissing  GrayReason = "hdr_metadata_missing"
	GrayTimescaleSuspect    GrayReason = "timescale_suspect"
	GrayDurationUnknown     GrayReason = "duration_unknown"
)

// grayReasons 是闭集，供校验与测试使用。
var grayReasons = map[GrayReason]struct{}{
	GrayVFRDetected:         {},
	GrayNoRollDistance:      {},
	GrayUnknownCodecPriming: {},
	GrayMediaRateNotOne:     {},
	GrayNoEdts:              {},
	GrayHDRMetadataMissing:  {},
	GrayTimescaleSuspect:    {},
	GrayDurationUnknown:     {},
}

// ValidGrayReason 报告 reason 是否在闭集内。
func ValidGrayReason(r GrayReason) bool {
	_, ok := grayReasons[r]
	return ok
}

// Value 是一个口径的原值（整数 + 单位），用于证据四元组。
type Value struct {
	Name  string `json:"name"` // 如 "stts_sum" / "mvhd_duration" / "elst_sum"
	Value int64  `json:"value"`
	Unit  string `json:"unit"` // "tick" / "sample" / "byte" / "deg" ...
}

// Delta 是差值（带单位）。fail 必填。
type Delta struct {
	Value int64  `json:"value"`
	Unit  string `json:"unit"`
}

// Finding 是一条判定结果。
type Finding struct {
	Module     string     `json:"module"`
	RuleID     string     `json:"rule_id"`
	Status     Status     `json:"status"`
	ByteOffset uint64     `json:"byte_offset,omitempty"`
	FieldPath  string     `json:"field_path,omitempty"`
	Values     []Value    `json:"values,omitempty"`
	Delta      *Delta     `json:"delta,omitempty"`
	GrayReason GrayReason `json:"gray_reason,omitempty"`
	Message    string     `json:"message"`
}

// ErrIncompleteEvidence 表示 fail 缺证据四元组。
var ErrIncompleteEvidence = errors.New("fail finding lacks complete evidence (byte_offset/field_path/values/delta)")

// Validate 校验单条 finding 的结构约束（渲染层在输出前必须调用）。
func (f Finding) Validate() error {
	switch f.Status {
	case Pass:
		return nil
	case Gray:
		if !ValidGrayReason(f.GrayReason) {
			return fmt.Errorf("finding %s/%s: gray_reason %q 不在闭集内（data/README §5.5）", f.Module, f.RuleID, f.GrayReason)
		}
		return nil
	case Fail:
		if f.ByteOffset == 0 && f.FieldPath == "" {
			// 偏移 0 是合法值，故两者都空才判缺
			return fmt.Errorf("%w: %s/%s", ErrIncompleteEvidence, f.Module, f.RuleID)
		}
		if f.FieldPath == "" {
			return fmt.Errorf("%w: %s/%s (field_path)", ErrIncompleteEvidence, f.Module, f.RuleID)
		}
		if len(f.Values) == 0 {
			return fmt.Errorf("%w: %s/%s (values)", ErrIncompleteEvidence, f.Module, f.RuleID)
		}
		if f.Delta == nil {
			return fmt.Errorf("%w: %s/%s (delta)", ErrIncompleteEvidence, f.Module, f.RuleID)
		}
		return nil
	default:
		return fmt.Errorf("finding %s/%s: unknown status %q", f.Module, f.RuleID, f.Status)
	}
}

// Summary 是三态计数（gray 不计入 fail）。
type Summary struct {
	Pass     int            `json:"pass"`
	Fail     int            `json:"fail"`
	Gray     int            `json:"gray"`
	ByModule map[string]int `json:"by_module_fail"`
}

// Report 是一次审计的完整结果。
type Report struct {
	Tool     string    `json:"tool"`
	Version  string    `json:"version"`
	File     string    `json:"file"` // basename only（不写绝对路径）
	Modules  []string  `json:"modules"`
	Findings []Finding `json:"findings"`
	Summary  Summary   `json:"summary"`
}

// SortFindings 用显式排序键保证确定性（不用 map 迭代顺序）。
func SortFindings(fs []Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.Module != b.Module {
			return a.Module < b.Module
		}
		if a.ByteOffset != b.ByteOffset {
			return a.ByteOffset < b.ByteOffset
		}
		if a.FieldPath != b.FieldPath {
			return a.FieldPath < b.FieldPath
		}
		return a.RuleID < b.RuleID
	})
}

// BuildSummary 计算三态计数。gray 明确不计入 fail。
func BuildSummary(fs []Finding) Summary {
	s := Summary{ByModule: map[string]int{}}
	for _, f := range fs {
		switch f.Status {
		case Pass:
			s.Pass++
		case Fail:
			s.Fail++
			s.ByModule[f.Module]++
		case Gray:
			s.Gray++
		}
	}
	return s
}

// ValidateAll 校验并排序，返回可渲染的 Report。
func (r *Report) ValidateAll() error {
	for _, f := range r.Findings {
		if err := f.Validate(); err != nil {
			return err
		}
	}
	SortFindings(r.Findings)
	r.Summary = BuildSummary(r.Findings)
	return nil
}

// Fingerprint 返回一条 finding 的稳定指纹，用于 JSON/Markdown 同源对拍。
func (f Finding) Fingerprint() string {
	return strings.Join([]string{f.Module, f.RuleID, string(f.Status), f.FieldPath, fmt.Sprint(f.ByteOffset)}, "|")
}

// VerdictLine 生成首屏第一段（字面固定）。
func (r Report) VerdictLine() string {
	if r.Summary.Fail > 0 {
		mods := make([]string, 0, len(r.Summary.ByModule))
		for m, n := range r.Summary.ByModule {
			if n > 0 {
				mods = append(mods, fmt.Sprintf("%s=%d", m, n))
			}
		}
		sort.Strings(mods)
		return fmt.Sprintf("VERDICT: FAIL (%d findings: %s, gray=%d)", r.Summary.Fail, strings.Join(mods, " "), r.Summary.Gray)
	}
	if r.Summary.Gray > 0 {
		return fmt.Sprintf("VERDICT: INCONCLUSIVE (%d gray findings)", r.Summary.Gray)
	}
	return "VERDICT: PASS (0 findings)"
}

// FirstLine 生成首屏第二段（偏移 + 字段路径；无 fail 时给出 gray 或占位）。
func (r Report) FirstLine() string {
	for _, f := range r.Findings {
		if f.Status == Fail || f.Status == Gray {
			d := "n/a"
			if f.Delta != nil {
				d = fmt.Sprintf("delta=%d %s", f.Delta.Value, f.Delta.Unit)
			}
			return fmt.Sprintf("FIRST: %s @0x%08X (%s, %s)", f.FieldPath, f.ByteOffset, f.RuleID, d)
		}
	}
	return "FIRST: none"
}

// RenderMarkdown 渲染人读版；与 JSON 同源（同一 Report）。
func (r Report) RenderMarkdown() string {
	var b strings.Builder
	b.WriteString(r.VerdictLine() + "\n")
	b.WriteString(r.FirstLine() + "\n")
	for _, f := range r.Findings {
		if f.Status == Pass {
			continue
		}
		reason := ""
		if f.GrayReason != "" {
			reason = " reason=" + string(f.GrayReason)
		}
		fmt.Fprintf(&b, "- [%s/%s] %s%s — %s\n", f.Module, f.RuleID, f.Status, reason, f.Message)
		if f.Status == Fail {
			fmt.Fprintf(&b, "  offset=0x%08X path=%s\n", f.ByteOffset, f.FieldPath)
			for _, v := range f.Values {
				fmt.Fprintf(&b, "  %s=%d %s\n", v.Name, v.Value, v.Unit)
			}
			if f.Delta != nil {
				fmt.Fprintf(&b, "  delta=%d %s\n", f.Delta.Value, f.Delta.Unit)
			}
		}
	}
	return b.String()
}
