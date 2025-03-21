// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package v3

import (
	json "encoding/json"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjsonCc79d2bdDecodeGithubComMysteriumnetworkDiscoveryProposalV3(in *jlexer.Lexer, out *Metadata) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "provider_id":
			out.ProviderID = string(in.String())
		case "service_type":
			out.ServiceType = string(in.String())
		case "country":
			out.Country = string(in.String())
		case "isp":
			out.ISP = string(in.String())
		case "ip_type":
			out.IPType = string(in.String())
		case "whitelist":
			out.Whitelist = bool(in.Bool())
		case "monitoring_failed":
			out.MonitoringFailed = bool(in.Bool())
		case "updated_at":
			if data := in.Raw(); in.Ok() {
				in.AddError((out.UpdatedAt).UnmarshalJSON(data))
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonCc79d2bdEncodeGithubComMysteriumnetworkDiscoveryProposalV3(out *jwriter.Writer, in Metadata) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"provider_id\":"
		out.RawString(prefix[1:])
		out.String(string(in.ProviderID))
	}
	{
		const prefix string = ",\"service_type\":"
		out.RawString(prefix)
		out.String(string(in.ServiceType))
	}
	if in.Country != "" {
		const prefix string = ",\"country\":"
		out.RawString(prefix)
		out.String(string(in.Country))
	}
	if in.ISP != "" {
		const prefix string = ",\"isp\":"
		out.RawString(prefix)
		out.String(string(in.ISP))
	}
	if in.IPType != "" {
		const prefix string = ",\"ip_type\":"
		out.RawString(prefix)
		out.String(string(in.IPType))
	}
	if in.Whitelist {
		const prefix string = ",\"whitelist\":"
		out.RawString(prefix)
		out.Bool(bool(in.Whitelist))
	}
	if in.MonitoringFailed {
		const prefix string = ",\"monitoring_failed\":"
		out.RawString(prefix)
		out.Bool(bool(in.MonitoringFailed))
	}
	{
		const prefix string = ",\"updated_at\":"
		out.RawString(prefix)
		out.Raw((in.UpdatedAt).MarshalJSON())
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v Metadata) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonCc79d2bdEncodeGithubComMysteriumnetworkDiscoveryProposalV3(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v Metadata) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonCc79d2bdEncodeGithubComMysteriumnetworkDiscoveryProposalV3(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *Metadata) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonCc79d2bdDecodeGithubComMysteriumnetworkDiscoveryProposalV3(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *Metadata) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonCc79d2bdDecodeGithubComMysteriumnetworkDiscoveryProposalV3(l, v)
}
