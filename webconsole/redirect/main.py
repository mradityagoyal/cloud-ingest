import webapp2

application = webapp2.WSGIApplication([
    webapp2.Route('/', webapp2.RedirectHandler, defaults={'_uri':'http://console.cloud.google.com/storage/transfers/on-premises'}),
], debug=False)
