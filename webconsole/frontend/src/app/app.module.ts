import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { NgModule } from '@angular/core';
import { HttpClientModule, HTTP_INTERCEPTORS } from '@angular/common/http';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { AngularMaterialImporterModule } from './angular-material-importer/angular-material-importer.module';
import { AppComponent } from './app.component';
import { AuthInterceptor } from './auth/auth-interceptor';
import { AuthService } from './auth/auth.service';
import { AppRoutingModule } from './app-routing.module';
import { BandwidthControlModule } from './bandwidth-control/bandwidth-control.module';
import { JobsModule } from './jobs/jobs.module';
import { ProjectSelectComponent } from './project-select/project-select.component';
import { LogsComponent } from './logs/logs.component';
import { ErrorDialogModule } from './util/error-dialog/error-dialog.module';
import { AgentComponent } from './agent/agent.component';
import { BandwidthControlComponent } from './bandwidth-control/bandwidth-control.component';

@NgModule({
  declarations: [
    AppComponent,
    ProjectSelectComponent,
    LogsComponent,
    AgentComponent,
    BandwidthControlComponent
  ],
  imports: [
    BrowserModule,
    BrowserAnimationsModule,
    HttpClientModule,
    FormsModule,
    ReactiveFormsModule,
    AngularMaterialImporterModule,
    BandwidthControlModule,
    JobsModule,
    ErrorDialogModule,
    AppRoutingModule
  ],
  providers: [
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
