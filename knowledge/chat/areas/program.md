<!-- status: draft -->
# Area guide: program

- Two levels: stages, and rows under a stage (activities, milestones, hold points). A new row needs parent_id, the stage's id.
- "PC" here is practical completion.
- Durations are working days. Send what the PM said in the duration field. Dates are computed by SiteWise; never send a date the PM did not give.
- Several rows alike are one add with labels, a list of names. Rows with the same name are numbered ("DA drawings 1" to "DA drawings 5").
- "B can't start until A finishes" is link_add with pred_id A and succ_id B.
- Releasing a hold point is the PM's to confirm.
