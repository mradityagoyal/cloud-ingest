# Ingest Web Console - Front-End
The front-end of the Ingest Web Console is built with Angular2 and Typescript.
[Angular CLI](https://cli.angular.io/) is used to serve, test, and lint the code.
In order to use the Ingest Web Console front-end, you need to have the Ingest Web Console
back-end up and running. See the back-end directory for instructions.

## Environment Set-up
1. Follow the [instructions to install](https://github.com/angular/angular-cli#installation) Angular CLI and its prerequisites.
2. Inside cloud-ingest/webconsole/front-end, run "npm install" to install the necessary
node modules (npm install uses the package.json file).

## Local Deployment
Run `ng serve`

If your local back-end is not at localhost:8080, update `apiUrl` in `src/environments/environment.ts`.

## Angular CLI Info
This project was generated with [Angular CLI](https://github.com/angular/angular-cli) version 1.2.1.

### Development server

Run `ng serve` for a dev server. Navigate to `http://localhost:4200/`. The app will automatically reload if you change any of the source files.

### Build

Run `ng build` to build the project. The build artifacts will be stored in the `dist/` directory. Use the `-prod` flag for a production build.

### Running unit tests

Run `ng test` to execute the unit tests via [Karma](https://karma-runner.github.io).

### Running end-to-end tests

Run `ng e2e` to execute the end-to-end tests via [Protractor](http://www.protractortest.org/).
Before running the tests make sure you are serving the app via `ng serve`.

### Development Tips
#### Code scaffolding

You can use `ng generate` to generate code skeletons.
For example, run `ng generate component new-feature` to generate a
new component named new-feature. You can also use
`ng generate directive|pipe|service|class|module`.
To learn more about the different types, read about them on the
[official Angular site](https://angular.io/guide/architecture).

### Further help

To get more help on the Angular CLI use `ng help` or go check out the
[Angular CLI README](https://github.com/angular/angular-cli/blob/master/README.md).


## App Engine Deployment
1. Make sure you have installed the [Google Cloud SDK](https://cloud.google.com/sdk/docs/)
2. Update `src/environments/environment.prod.ts` with the url of your back-end.
  For example:
  ```python
  export const environment = {
    production: true,
    apiUrl: 'http://myprodbackend.wow:8080'
  }
  ```
3. Run `ng build --prod --env=prod`  
  The --prod enables ahead of time compilation, and the --env=prod flag tells Angular CLI to use the prod environment file.
4. Run `gcloud app deploy --project <your-cloud-project-id>` with the id of your Google Cloud project.


