# Velero Status Restorer Plugin

Velero by default flushes the status of backed-up objects.

This plugin applies the status of selcted CRs after they were restored from velero.

This functionality is implemented in a Restore Item Action.
