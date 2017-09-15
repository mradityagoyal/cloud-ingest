import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { HttpClientModule, HTTP_INTERCEPTORS } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { AngularMaterialImporterModule } from './angular-material-importer.module';

import { AppComponent } from './app.component';
import { JobsService } from './jobs.service';
import { AuthInterceptor } from './auth-interceptor';
import { AuthService } from './auth.service';
import { AppRoutingModule } from './app-routing.module'
import { JobConfigsComponent } from './job-configs.component';
import { JobConfigAddDialogComponent } from './job-config-add-dialog.component';

import { JobRunsModule } from './job-runs/job-runs.module'

@NgModule({
  declarations: [
    AppComponent, JobConfigsComponent,
    JobConfigAddDialogComponent
  ],
  entryComponents: [JobConfigAddDialogComponent],
  imports: [
    BrowserModule,
    BrowserAnimationsModule,
    HttpClientModule,
    FormsModule,
    AngularMaterialImporterModule,
    JobRunsModule,
    AppRoutingModule
  ],
  providers: [
    JobsService,
    AuthService,
    {
      provide: HTTP_INTERCEPTORS,
      useClass: AuthInterceptor,
      multi: true,
    },
  ],
  bootstrap: [AppComponent]
})
export class AppModule { }
