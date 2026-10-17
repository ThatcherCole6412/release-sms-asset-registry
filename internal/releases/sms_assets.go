package releases

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"example.com/release-sms-registry/internal/infrai"
)

type Registry interface {
	CreateSignature(context.Context, infrai.SignatureInput, string) (infrai.CreatedAsset, error)
	CreateTemplate(context.Context, infrai.TemplateInput, string) (infrai.CreatedAsset, error)
}

type ReleaseInput struct {
	Product    string `json:"product"`
	ReleaseID  string `json:"release_id"`
	BuildEvent string `json:"build_event"`
	Operation  string `json:"operation"`
	Diagnostic string `json:"diagnostic"`
	Signature  string `json:"signature"`
	ProofURL   string `json:"proof_url,omitempty"`
}

type AssetPlan struct {
	Key  string
	Body string
	Vars []string
}

type Registration struct {
	ReleaseID string                `json:"release_id"`
	Selected  string                `json:"selected_template"`
	Signature infrai.CreatedAsset   `json:"signature"`
	Templates []infrai.CreatedAsset `json:"templates"`
}

var cleanName = regexp.MustCompile(`[^a-z0-9]+`)

func SelectTemplate(input ReleaseInput) string {
	if input.Diagnostic == "critical" {
		return "diagnostic-critical"
	}
	if input.BuildEvent == "failed" {
		return "build-failed"
	}
	return "release-promoted"
}

func Register(ctx context.Context, registry Registry, input ReleaseInput) (Registration, error) {
	namespace := slug(input.Product + "-" + input.ReleaseID)
	result := Registration{ReleaseID: input.ReleaseID, Selected: SelectTemplate(input)}

	signature, err := registry.CreateSignature(ctx, infrai.SignatureInput{
		Name: input.Signature, Type: "company", ProofURL: input.ProofURL,
		Remark: "Developer release notifications for " + input.Product,
	}, namespace+":signature")
	if err != nil {
		return result, fmt.Errorf("register signature: %w", err)
	}
	result.Signature = signature

	plans := []AssetPlan{
		{Key: "build-failed", Body: "{{product}} build {{build_id}} failed. Diagnostic: {{diagnostic}}", Vars: []string{"product", "build_id", "diagnostic"}},
		{Key: "release-promoted", Body: "{{product}} release {{release_id}} was promoted to {{environment}}.", Vars: []string{"product", "release_id", "environment"}},
		{Key: "diagnostic-critical", Body: "Critical {{product}} diagnostic {{diagnostic_id}}: {{summary}}", Vars: []string{"product", "diagnostic_id", "summary"}},
	}
	for _, plan := range plans {
		asset, err := registry.CreateTemplate(ctx, infrai.TemplateInput{
			Name: namespace + "-" + plan.Key, Body: plan.Body, Locale: "en-US", TemplateVars: plan.Vars,
		}, namespace+":template:"+plan.Key)
		if err != nil {
			return result, fmt.Errorf("register %s template: %w", plan.Key, err)
		}
		result.Templates = append(result.Templates, asset)
	}
	return result, nil
}

func Validate(input ReleaseInput) error {
	if strings.TrimSpace(input.Product) == "" || strings.TrimSpace(input.ReleaseID) == "" || strings.TrimSpace(input.Signature) == "" {
		return fmt.Errorf("product, release_id, and signature are required")
	}
	return nil
}

func slug(value string) string {
	value = strings.Trim(cleanName.ReplaceAllString(strings.ToLower(value), "-"), "-")
	if value == "" {
		return "release"
	}
	return value
}
