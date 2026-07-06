# Prioritisation Rubric

The `/now` prioritisation engine is implemented as a transparent, banded model:

```text
high | medium | low
```

Signals include:

- urgency;
- blocker resolution value;
- release impact;
- recency;
- temporary focus match;
- fit for available time;
- uncertainty penalty;
- dependency penalty.

The default `config.yml` weights are:

```yaml
priority_weights:
  urgency: medium
  blocker_resolution_value: medium
  release_impact: medium
  recency: low
  user_focus_match: medium
  fit_for_available_time: medium
  uncertainty_penalty: medium
  dependency_penalty: medium
```

Temporary CLI inputs such as `--focus` and `--minutes` affect only the current recommendation and are not persisted.
