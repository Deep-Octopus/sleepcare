# Phase-one access control

## Responsibility

This module is the final reconciliation layer for phase-one employee access. Individual business modules still own their APIs, menus, buttons, DataScope-aware queries, and responsibility checks; the reconciliation layer runs after those modules and makes the four fixed employee roles converge on one explicit allowlist.

It does not replace JWT, Casbin, DataScope, or domain authorization. A request succeeds only when every applicable layer allows it.

## Entry points

- Stable domain-role constants: `server/model/careclient/constants.go`
- Domain decisions: `server/internal/accesspolicy/`
- Metadata reconciliation: `server/initialize/phase_one_access_control.go`
- Boot order: the final care initializer in `server/initialize/gorm.go`
- Dynamic route components: `web/src/view/sleep-care/**/detail.vue`
- Generated component-name map: `web/src/pathInfo.json`

## Role boundaries

- `CARE_STEWARD`: responsibility-scoped clients, runtime plans, tasks, cases, deliveries, and plan-definition preview.
- `CLINICIAN`: responsibility-scoped runtime data plus questionnaire and plan-definition preview.
- `SUPERVISOR`: department-tree runtime reads, explicitly allowed maintenance/case actions, supervision, and content preview.
- `CONTENT_ADMIN`: questionnaire and plan-definition preview only; no client or runtime access.
- System role `888`: may configure RBAC but receives no care-domain menu, button, or `/care/**` policy from this module.
- Client sessions: remain on the isolated mobile route and middleware chain and never receive employee menus.

`CONTENT_ADMIN` is mapped through the existing `CareAuthorityProfile`. Plan-content access uses `ResolvePlanContent`; runtime services continue to use `ResolveCareClient`. This split is required so content visibility cannot grant access to clients, plans in execution, tasks, cases, deliveries, or supervision records.

## Menu and route invariants

The employee tree is:

```text
SleepCare
├── CareWorkbench
├── CareClients
├── CareExecution
│   ├── CareTasks
│   ├── CareAttentionCases
│   └── CareDeliveries
├── CareContent
│   ├── CareQuestionnaires
│   └── CarePlans
└── CareSupervision
    ├── CareDailySummaries
    └── CareReviewQueue
```

Four detail routes are hidden metadata, not authorization boundaries:

- `CareClientDetail` highlights `CareClients`.
- `CareTaskDetail` highlights `CareTasks`.
- `CareAttentionCaseDetail` highlights `CareAttentionCases`.
- `CareReviewDetail` highlights `CareReviewQueue`.

Each wrapper component has a stable unique `defineOptions({ name })` and opens the existing detail drawer from the route parameter. A role receives a hidden route only when it receives the corresponding list capability.

Default routes must be visible authorized leaves: steward and clinician use `CareWorkbench`, supervisor uses `CareDailySummaries`, and content administrator uses `CareQuestionnaires`.

## Reconciliation rules

- Menu, button, and Casbin links for the four fixed roles are replaced from explicit allowlists on every enabled fixture boot.
- Stale policies for those roles are removed before the current shell and business policies are inserted.
- Care-domain grants accidentally attached to role `888` are removed.
- Menu hierarchy and hidden-route metadata are maintained even when fixed fixtures are disabled; role/user/grant reconciliation remains disabled.
- Reconciliation is idempotent and does not create or alter business lifecycle records.

## Required negative evidence

- Every role/API pair in the phase-one union is tested as allowed or denied.
- Every role/menu and role/button pair is tested against the allowlist.
- Default routes are tested for grant, visibility, and leaf status.
- Missing identity, system identity, unmapped role, content-role runtime access, same-department missing responsibility, and cross-organization access fail closed.
- Frontend route absence or button hiding is never cited as the backend denial mechanism.
