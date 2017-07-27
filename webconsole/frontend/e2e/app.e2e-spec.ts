import { WebconsoleFrontEndPage } from './app.po';

describe('webconsole-front-end App', () => {
  let page: WebconsoleFrontEndPage;

  beforeEach(() => {
    page = new WebconsoleFrontEndPage();
  });

  it('should display title', () => {
    page.navigateTo();
    expect(page.getDisplayedTitleText()).toEqual('Ingest Web Console');
  });
});
