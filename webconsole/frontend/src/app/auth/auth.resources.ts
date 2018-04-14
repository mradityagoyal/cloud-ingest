/**
 * UserProfile represents the authenticated Google user profile information.
 */
export interface UserProfile {
  Name: string;
}

export interface Policy {
  bindings: Binding[];
}

export interface UpdatePolicyRequest {
  policy: Policy;
}

export interface PolicyResponse {
  result: Policy;
}

export interface Binding {
  role: string;
  members: string[];
}
