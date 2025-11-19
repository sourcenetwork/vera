package types

// IsExpiredAgainst return true if c is expired when taken ts as the target time
func (c *RegistrationsCommitment) IsExpiredAgainst(ts *Timestamp) (bool, error) {
	return c.Metadata.CreationTs.IsAfter(c.Validity, ts)
}
