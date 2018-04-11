/**
 * @fileoverview This file contains a module that imports all of the angular
 *     material (https://material.angular.io/) components used in this
 *     application. This way, all of the angular material imports are kept in
 *     one place.
 */
import { NgModule } from '@angular/core';
import { MatAutocompleteModule,
        MatButtonModule,
        MatButtonToggleModule,
        MatCardModule,
        MatCheckboxModule,
        MatDialogModule,
        MatExpansionModule,
        MatIconModule,
        MatInputModule,
        MatListModule,
        MatProgressSpinnerModule,
        MatSidenavModule,
        MatSnackBarModule,
        MatTableModule,
        MatToolbarModule,
        MatTooltipModule,
        MatTabsModule } from '@angular/material';

import { CdkTableModule } from '@angular/cdk/table';

@NgModule({
  imports: [MatAutocompleteModule,
                MatButtonModule,
                MatButtonToggleModule,
                MatCardModule,
                MatCheckboxModule,
                MatDialogModule,
                MatExpansionModule,
                MatIconModule,
                MatInputModule,
                MatListModule,
                MatProgressSpinnerModule,
                MatSidenavModule,
                MatSnackBarModule,
                MatTableModule,
                MatToolbarModule,
                MatTooltipModule,
                MatTabsModule, ],
  exports: [MatAutocompleteModule,
                MatButtonModule,
                MatButtonToggleModule,
                MatCardModule,
                MatCheckboxModule,
                MatDialogModule,
                MatExpansionModule,
                MatIconModule,
                MatInputModule,
                MatListModule,
                MatProgressSpinnerModule,
                MatSidenavModule,
                MatSnackBarModule,
                MatTableModule,
                MatToolbarModule,
                MatTooltipModule,
                MatTabsModule]
})

export class AngularMaterialImporterModule { }
