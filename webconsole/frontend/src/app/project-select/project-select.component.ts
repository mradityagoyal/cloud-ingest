


import { HttpClient, HttpParams } from '@angular/common/http';
import { Component } from '@angular/core';
import { FormControl } from '@angular/forms';
import { NavigationExtras, Router } from '@angular/router';
import { Observable } from 'rxjs';
import { map, startWith } from 'rxjs/operators';

import { GoogleCloudApiProjectsResponse, GoogleCloudProject } from './project-select.resources';

@Component({
  selector: 'app-project-select',
  templateUrl: './project-select.component.html',
  styleUrls: ['./project-select.component.css']
})
export class ProjectSelectComponent {
  static GCP_RESOURCE_MANAGER_API_URL = 'https://cloudresourcemanager.googleapis.com/v1/projects';

  googleCloudProjects: GoogleCloudProject[] = [];
  filteredGoogleCloudProjects: Observable<GoogleCloudProject[]>;
  gcsProjectIdControl: FormControl = new FormControl();
  gcsProjectId: string;

  constructor(private router: Router,
              private http: HttpClient) {
    this.loadAllGoogleCloudProjects();
    this.setFilteredProjects();
  }

  /**
   * Loads all of the Google Cloud Projects into the google cloud projects variable. Navigates
   * through pages until there is no nextPage token.
   */
  private loadAllGoogleCloudProjects() {
    const appendNext = (nextPageToken?: string) => {
      let pageTokenParams: HttpParams = new HttpParams();
      if (nextPageToken != null) {
        pageTokenParams = pageTokenParams.set('pageToken', nextPageToken);
      }
      this.http.get(ProjectSelectComponent.GCP_RESOURCE_MANAGER_API_URL, {params: pageTokenParams}).subscribe(
        (response: GoogleCloudApiProjectsResponse) => {
          this.googleCloudProjects = this.googleCloudProjects.concat(response.projects);
          if (response.nextPageToken != null) {
            appendNext(response.nextPageToken);
          }
        });
    };
    appendNext();
  }

  /**
   * Sets the filteredGoogleCloudProjects variable to re-populate every time the user types on
   * the input box.
   */
  private setFilteredProjects() {
    this.filteredGoogleCloudProjects = this.gcsProjectIdControl.valueChanges.pipe(
      startWith(null),
      map(val => val ? this.filter(val) : this.googleCloudProjects.slice(0, 3)));
  }

  private filter(val: string): GoogleCloudProject[] {
    return this.googleCloudProjects.filter(project =>
      project.name.toLowerCase().indexOf(val.toLowerCase()) === 0 ||
      project.projectId.toLowerCase().indexOf(val.toLowerCase()) === 0);
   }

  onProjectSelectSubmit() {
    const navigationExtras: NavigationExtras = {
      queryParams: { project: this.gcsProjectId }
    };
    this.router.navigate(['/jobs'], navigationExtras);
  }
}
