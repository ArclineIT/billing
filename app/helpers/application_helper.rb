module ApplicationHelper
  # Stroke icons on a 24px grid, drawn with currentColor.
  ICONS = {
    "arrow" => '<path d="M5 12h14"/><path d="m13 6 6 6-6 6"/>',
    "check" => '<path d="m5 12 5 5 9-10"/>',
    "contrast" => '<circle cx="12" cy="12" r="9"/><path d="M12 3a9 9 0 0 1 0 18Z" fill="currentColor"/>',
    "download" => '<path d="M12 3v12"/><path d="m7 11 5 5 5-5"/><path d="M5 21h14"/>',
    "mail" => '<rect x="2" y="4" width="20" height="16" rx="2"/><path d="m22 7-10 6L2 7"/>',
    "menu" => '<path d="M4 7h16"/><path d="M4 12h16"/><path d="M4 17h16"/>'
  }.freeze

  def icon(name, size: 20, **options)
    tag.svg(ICONS.fetch(name.to_s).html_safe, xmlns: "http://www.w3.org/2000/svg", viewBox: "0 0 24 24",
      width: size, height: size, fill: "none", stroke: "currentColor", "stroke-width": 1.75,
      "stroke-linecap": "round", "stroke-linejoin": "round", "aria-hidden": true, focusable: false,
      class: [ "icon", options[:class] ])
  end

  def nav_link(label, path)
    current = request.path == path || request.path.start_with?("#{path}/")
    link_to label, path, class: "nav__link", aria: { current: ("page" if current) }
  end

  # Formats a Date/Time/string as e.g. "Jan 2, 2026", or an em dash when blank.
  def format_date(value)
    return "—" if value.blank?

    value.to_date.strftime("%b %-d, %Y")
  end
end
