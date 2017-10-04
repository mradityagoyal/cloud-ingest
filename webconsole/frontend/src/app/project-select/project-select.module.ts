import { NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

import { CdkTableModule } from '@angular/cdk/table';

@NgModule({
  imports: [FormsModule,
            ReactiveFormsModule,
            BrowserAnimationsModule],
  exports: [FormsModule,
            ReactiveFormsModule,
            BrowserAnimationsModule]
})

export class ProjectSelectModule { }
