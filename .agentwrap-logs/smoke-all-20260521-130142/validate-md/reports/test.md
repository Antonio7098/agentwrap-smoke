# Report

## Summary

Validated that the markdown template renders correctly with placeholders replaced by actual content. The template structure (h1, h2 sections) is preserved and content is successfully injected.

## Details

- Template: `template.md` — contains the base structure with `[PLACEHOLDER:]` markers
- Validation model: `opencode/deepseek-v4-flash-free`
- Run ID: `validation-2`
- Test: Markdown template placeholder substitution
- Result: All placeholders were identified and replaced with descriptive content
- The final output maintains the original heading hierarchy (`# Report` → `## Summary` → `## Details`)
