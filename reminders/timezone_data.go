package reminders

// Embed the IANA timezone database into the binary. The production image is
// based on Alpine and intentionally has no OS timezone package installed.
import _ "time/tzdata"
