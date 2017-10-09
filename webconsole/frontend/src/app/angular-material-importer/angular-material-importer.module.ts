/**
 * @fileoverview This file contains a module that imports all of the angular
 *     material (https://material.angular.io/) components used in this
 *     application. This way, all of the angular material imports are kept in
 *     one place.
 */
import { NgModule } from '@angular/core';
import { MatIconModule,
        MatSidenavModule,
        MatListModule,
        MatToolbarModule,
        MatCardModule,
        MatTooltipModule,
        MatButtonModule,
        MatDialogModule,
        MatInputModule,
        MatCheckboxModule,
        MatTableModule,
        MatProgressSpinnerModule,
        MatTabsModule,
        MatSnackBarModule,
        MatAutocompleteModule } from '@angular/material';

import { CdkTableModule } from '@angular/cdk/table';

@NgModule({
  imports: [MatIconModule,
            MatSidenavModule,
            MatListModule,
            MatToolbarModule,
            MatCardModule,
            MatTooltipModule,
            MatButtonModule,
            MatDialogModule,
            MatInputModule,
            MatCheckboxModule,
            MatTableModule,
            MatProgressSpinnerModule,
            CdkTableModule,
            MatTabsModule,
            MatAutocompleteModule],
  exports: [MatIconModule,
            MatSidenavModule,
            MatListModule,
            MatToolbarModule,
            MatCardModule,
            MatTooltipModule,
            MatButtonModule,
            MatDialogModule,
            MatInputModule,
            MatCheckboxModule,
            MatTableModule,
            MatProgressSpinnerModule,
            MatTabsModule,
            MatSnackBarModule,
            CdkTableModule,
            MatAutocompleteModule]
})

export class AngularMaterialImporterModule { }
