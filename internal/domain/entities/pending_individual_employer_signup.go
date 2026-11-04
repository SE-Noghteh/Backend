package entities

// A complete, validated employer submission awaiting phone verification.
// Registration contains password hashes and private object references, never raw passwords.
type PendingIndividualEmployerSignup = PendingEmployerSignup[IndividualEmployer]
