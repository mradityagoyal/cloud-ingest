export interface GoogleCloudProject {
  projectNumber: number;
  projectId: string;
  name: string;
}

/**
 * The api response for the project list, as defined by:
 * https://cloud.google.com/resource-manager/reference/rest/v1/projects/list
 */
export interface GoogleCloudApiProjectsResponse {
  projects: GoogleCloudProject[];
  nextPageToken?: string;
}
