# Extend the Go 'TaskJSON' struct in 'cmd/build/main.go' to include fields for 'approval_attempts' and 'lead_interventions', and update 'runTaskQuery' to query and scan these values from the tasks database.

The task requires modifying the Go CLI to return more complete metadata when querying tasks. Specifically, the 'TaskJSON' struct in 'cmd/build/main.go' must be extended to include two new fields: 'ApprovalAttempts' (JSON: 'approval_attempts') and 'LeadInterventions' (JSON: 'lead_interventions').

To populate these fields, the 'runTaskQuery' function must be modified. The SQL queries for 'next', 'blocked', and 'stuck' subcommands must be updated to retrieve the 'approval_attempts' and 'lead_interventions' columns. Finally, the row scanning logic must be updated to scan these queried values into the new struct fields before the struct is serialized to JSON and output.
