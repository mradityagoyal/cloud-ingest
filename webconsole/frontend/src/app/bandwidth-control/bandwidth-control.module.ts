import { NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { BandwidthControlService } from './bandwidth-control.service';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

import { CdkTableModule } from '@angular/cdk/table';

@NgModule({
  imports: [FormsModule,
            ReactiveFormsModule,
            BrowserAnimationsModule],
  exports: [FormsModule,
            ReactiveFormsModule,
            BrowserAnimationsModule],
  providers: [ BandwidthControlService ]
})

export class BandwidthControlModule { }
