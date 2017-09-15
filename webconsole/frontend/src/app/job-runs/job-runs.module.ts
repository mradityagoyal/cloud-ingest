import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

import { AngularMaterialImporterModule } from '../angular-material-importer.module';
import { JobsService } from '../jobs.service';
import { JobRunListComponent } from './job-run-list/job-run-list.component';
import { CreateRunComponent } from './create-run/create-run.component';

import { JobRunsRoutingModule } from './job-runs-routing.module'


@NgModule({
  imports: [
    CommonModule,
    FormsModule,
    BrowserModule,
    BrowserAnimationsModule,
    AngularMaterialImporterModule,
    JobRunsRoutingModule
  ],
  declarations: [
    JobRunListComponent,
    CreateRunComponent
  ],
  providers: [ JobsService ]
})
export class JobRunsModule { }
