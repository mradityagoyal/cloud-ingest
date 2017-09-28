import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { InfrastructureRoutingModule } from './infrastructure-routing.module';
import { InfrastructureComponent } from './infrastructure.component';
import { InfrastructureStatusItemComponent } from './infrastructure-status-item/infrastructure-status-item.component';

import { AngularMaterialImporterModule } from '../angular-material-importer/angular-material-importer.module';
import { InfrastructureService } from './infrastructure.service';


@NgModule({
  imports: [
    CommonModule,
    BrowserModule,
    BrowserAnimationsModule,
    AngularMaterialImporterModule,
    InfrastructureRoutingModule
  ],
  declarations: [
    InfrastructureStatusItemComponent,
    InfrastructureComponent
  ],
  providers: [
    InfrastructureService,
  ]
})
export class InfrastructureModule { }
