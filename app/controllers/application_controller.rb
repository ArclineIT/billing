class ApplicationController < ActionController::Base
  include Authentication

  # Allow webhook and other machine-to-machine endpoints to skip the browser gate.
  before_action :restrict_to_modern_browsers

  # Changes to the importmap will invalidate the etag for HTML responses
  stale_when_importmap_changes

  private

  def restrict_to_modern_browsers
    allow_browser versions: :modern, block: :render_unsupported_browser
  end

  def render_unsupported_browser
    render file: Rails.root.join("public/406-unsupported-browser.html"),
           layout: false, status: :not_acceptable
  end
end
