#!/usr/bin/env ruby
# Local adapter for jimmyjazz/IDEF0-SVG. Keep the upstream layout code intact.
require 'cgi'
require 'json'
require 'optparse'
require_relative 'upstream/lib/idef0/cli'
require_relative 'language'
require_relative 'boundary'

module CareeriaIdef0Style
  COLOR = /\A(?:\#[0-9a-fA-F]{3}(?:[0-9a-fA-F]{3})?|[a-zA-Z]+)\z/
  DEFAULT_KEYS = %w[arrow arrow_text block_fill block_stroke block_text].freeze
  BLOCK_KEYS = %w[fill stroke text].freeze
  ARROW_KEYS = %w[stroke text].freeze

  class << self
    attr_accessor :settings

    def validate_colors(values, keys, location)
      raise "#{location} must be an object" unless values.is_a?(Hash)
      values.each do |key, value|
        raise "unknown #{location} field: #{key}" unless keys.include?(key)
        raise "invalid #{location}.#{key} color: #{value.inspect}" unless value.is_a?(String) && COLOR.match?(value)
      end
    end

    def load(path)
      data = path ? JSON.parse(File.read(path)) : {}
      raise 'style file must be an object' unless data.is_a?(Hash)
      unknown = data.keys - %w[mode process views defaults blocks arrows boundaries order branching corners crossings]
      raise "unknown style fields: #{unknown.join(', ')}" unless unknown.empty?
      if data.key?('order')
        order = data['order']
        raise "invalid order mode: #{order.inspect}" unless %w[auto source fixed heuristic].include?(order)
      end
      if data.key?('branching')
        branching = data['branching']
        valid = [true, false, 'auto', 'true', 'false', 'none', 'mechanisms', 'all']
        raise "invalid branching mode: #{branching.inspect}" unless valid.include?(branching)
      end
      if data.key?('corners')
        corners = data['corners']
        raise "invalid corners mode: #{corners.inspect}" unless %w[rounded sharp].include?(corners)
      end
      if data.key?('crossings')
        crossings = data['crossings']
        valid_cr = [true, false, 'arc', 'bridge', 'gap', 'break', 'none', 'plain', 'true', 'false']
        raise "invalid crossings mode: #{crossings.inspect}" unless valid_cr.include?(crossings)
      end
      validate_colors(data.fetch('defaults', {}), DEFAULT_KEYS, 'defaults')
      { 'blocks' => BLOCK_KEYS, 'arrows' => ARROW_KEYS }.each do |group, keys|
        entries = data.fetch(group, {})
        raise "#{group} must be an object" unless entries.is_a?(Hash)
        entries.each { |name, values| validate_colors(values, keys, "#{group}.#{name}") }
      end
      CareeriaIdef0Boundary.load(data)
      self.settings = data
    end

    def order_mode
      settings&.fetch('order', 'auto') || 'auto'
    end

    def branching_mode
      settings&.fetch('branching', 'auto')
    end

    def branching_mechanisms?
      mode = branching_mode
      return false if mode == false || mode == 'false' || mode == 'none'
      true
    end

    def branching_controls?
      mode = branching_mode
      return false if mode == false || mode == 'false' || mode == 'none' || mode == 'mechanisms'
      true
    end

    def corners_mode
      settings&.fetch('corners', 'rounded') || 'rounded'
    end

    def crossings_mode
      raw = settings&.fetch('crossings', 'arc')
      raw = 'arc' if raw.nil? || raw == true || raw == 'true'
      case raw
      when 'bridge' then 'arc'
      when 'break' then 'gap'
      when 'plain', false, 'false' then 'none'
      else raw.to_s
      end
    end

    def box_svg(box)
      defaults = settings.fetch('defaults', {})
      style = settings.fetch('blocks', {}).fetch(box.name, {})
      fill = style.fetch('fill', defaults.fetch('block_fill', 'none'))
      stroke = style.fetch('stroke', defaults.fetch('block_stroke', 'black'))
      text = style.fetch('text', defaults.fetch('block_text', 'black'))
      markup = box.to_svg
        .sub("fill='none' stroke='black'", "fill='#{fill}' stroke='#{stroke}'")
        .sub('<text ', "<text fill='#{text}' ")
      "<g data-idef0-block='#{CGI.escapeHTML(box.name)}'>#{markup}</g>"
    end

    def arrow_svg(line)
      defaults = settings.fetch('defaults', {})
      style = settings.fetch('arrows', {}).fetch(line.name, {})
      stroke = style.fetch('stroke', defaults.fetch('arrow', 'black'))
      text = style.fetch('text', defaults.fetch('arrow_text', 'black'))
      markup = line.to_svg
        .gsub("stroke='black'", "stroke='#{stroke}'")
        .gsub("fill='black'", "fill='#{stroke}'")
        .gsub('<text ', "<text fill='#{text}' ")
      attributes, extras = CareeriaIdef0Boundary.decorate(line, stroke)
      "<g data-idef0-arrow='#{CGI.escapeHTML(line.name)}'#{attributes}>#{markup}#{extras}</g>"
    end

    def branched_mechanism_svg(name, lines, bus_index)
      defaults = settings.fetch('defaults', {})
      style = settings.fetch('arrows', {}).fetch(name, {})
      stroke = style.fetch('stroke', defaults.fetch('arrow', 'black'))
      text_color = style.fetch('text', defaults.fetch('arrow_text', 'black'))

      sorted = lines.sort_by(&:x1)
      primary = sorted.first
      x_min = sorted.first.x1
      x_max = sorted.last.x1
      y_bottom = sorted.map(&:y1).max
      boxes_bottom = sorted.map(&:y2).max

      bus_y = boxes_bottom + 15 + bus_index * 20
      rounded = corners_mode == 'rounded'
      r = 10
      cp = 5

      first_target_y = sorted.first.y2
      trunk = %(<line x1='#{x_min}' y1='#{y_bottom - 20}' x2='#{x_min}' y2='#{first_target_y}' stroke='#{stroke}' />\n) \
        + %(<polygon fill='#{stroke}' stroke='#{stroke}' points='#{x_min},#{first_target_y} #{x_min - 3},#{first_target_y + 6} #{x_min + 3},#{first_target_y + 6} #{x_min},#{first_target_y}' />)

      if rounded && (x_max - x_min) >= r * 2
        last_line = sorted.last
        last_target_y = last_line.y2
        bus_and_last = %(<path fill='none' stroke='#{stroke}' d='M #{x_min} #{bus_y + r} C #{x_min} #{bus_y + cp} #{x_min + cp} #{bus_y} #{x_min + r} #{bus_y} L #{x_max - r} #{bus_y} C #{x_max - cp} #{bus_y} #{x_max} #{bus_y - cp} #{x_max} #{bus_y - r} L #{x_max} #{last_target_y}' />\n) \
          + %(<polygon fill='#{stroke}' stroke='#{stroke}' points='#{x_max},#{last_target_y} #{x_max - 3},#{last_target_y + 6} #{x_max + 3},#{last_target_y + 6} #{x_max},#{last_target_y}' />)

        other_branches = sorted[1...-1].map do |line|
          x = line.x1
          target_y = line.y2
          %(<path fill='none' stroke='#{stroke}' d='M #{x - r} #{bus_y} C #{x - cp} #{bus_y} #{x} #{bus_y - cp} #{x} #{bus_y - r} L #{x} #{target_y}' />\n) \
            + %(<polygon fill='#{stroke}' stroke='#{stroke}' points='#{x},#{target_y} #{x - 3},#{target_y + 6} #{x + 3},#{target_y + 6} #{x},#{target_y}' />)
        end.join("\n")

        branches_markup = other_branches.empty? ? bus_and_last : "#{bus_and_last}\n#{other_branches}"
      else
        bus = %(<line x1='#{x_min}' y1='#{bus_y}' x2='#{x_max}' y2='#{bus_y}' stroke='#{stroke}' />)
        other_branches = sorted[1..-1].map do |line|
          x = line.x1
          target_y = line.y2
          %(<line x1='#{x}' y1='#{bus_y}' x2='#{x}' y2='#{target_y}' stroke='#{stroke}' />\n) \
            + %(<polygon fill='#{stroke}' stroke='#{stroke}' points='#{x},#{target_y} #{x - 3},#{target_y + 6} #{x + 3},#{target_y + 6} #{x},#{target_y}' />)
        end.join("\n")
        branches_markup = "#{bus}\n#{other_branches}"
      end

      label_svg = %(<text fill='#{text_color}' text-anchor='middle' x='#{x_min}' y='#{y_bottom - 5}'>#{CGI.escapeHTML(name)}</text>)
      attributes, extras = CareeriaIdef0Boundary.decorate(primary, stroke)

      sorted[1..-1].each do |other_line|
        _other_attr, other_extra = CareeriaIdef0Boundary.decorate(other_line, stroke)
        if other_extra.include?('data-idef0-tunnel-marker')
          extras << other_extra.scan(/<g data-idef0-tunnel-marker='parent'.*?<\/g>/m).join
        end
      end

      markup = "#{trunk}\n#{branches_markup}\n#{label_svg}"
      "<g data-idef0-arrow='#{CGI.escapeHTML(name)}'#{attributes}>\n#{markup}\n#{extras}\n</g>"
    end

    def branched_guidance_svg(name, lines, bus_index)
      defaults = settings.fetch('defaults', {})
      style = settings.fetch('arrows', {}).fetch(name, {})
      stroke = style.fetch('stroke', defaults.fetch('arrow', 'black'))
      text_color = style.fetch('text', defaults.fetch('arrow_text', 'black'))

      sorted = lines.sort_by(&:x1)
      primary = sorted.first
      x_min = sorted.first.x1
      x_max = sorted.last.x1
      y_top = sorted.map(&:y1).min
      boxes_top = sorted.map(&:y2).min

      bus_y = boxes_top - 15 - bus_index * 20
      rounded = corners_mode == 'rounded'
      r = 10
      cp = 5

      first_target_y = sorted.first.y2
      trunk = %(<line x1='#{x_min}' y1='#{y_top + 20}' x2='#{x_min}' y2='#{first_target_y}' stroke='#{stroke}' />\n) \
        + %(<polygon fill='#{stroke}' stroke='#{stroke}' points='#{x_min},#{first_target_y} #{x_min - 3},#{first_target_y - 6} #{x_min + 3},#{first_target_y - 6} #{x_min},#{first_target_y}' />)

      if rounded && (x_max - x_min) >= r * 2
        last_line = sorted.last
        last_target_y = last_line.y2
        bus_and_last = %(<path fill='none' stroke='#{stroke}' d='M #{x_min} #{bus_y - r} C #{x_min} #{bus_y - cp} #{x_min + cp} #{bus_y} #{x_min + r} #{bus_y} L #{x_max - r} #{bus_y} C #{x_max - cp} #{bus_y} #{x_max} #{bus_y + cp} #{x_max} #{bus_y + r} L #{x_max} #{last_target_y}' />\n) \
          + %(<polygon fill='#{stroke}' stroke='#{stroke}' points='#{x_max},#{last_target_y} #{x_max - 3},#{last_target_y - 6} #{x_max + 3},#{last_target_y - 6} #{x_max},#{last_target_y}' />)

        other_branches = sorted[1...-1].map do |line|
          x = line.x1
          target_y = line.y2
          %(<path fill='none' stroke='#{stroke}' d='M #{x - r} #{bus_y} C #{x - cp} #{bus_y} #{x} #{bus_y + cp} #{x} #{bus_y + r} L #{x} #{target_y}' />\n) \
            + %(<polygon fill='#{stroke}' stroke='#{stroke}' points='#{x},#{target_y} #{x - 3},#{target_y - 6} #{x + 3},#{target_y - 6} #{x},#{target_y}' />)
        end.join("\n")

        branches_markup = other_branches.empty? ? bus_and_last : "#{bus_and_last}\n#{other_branches}"
      else
        bus = %(<line x1='#{x_min}' y1='#{bus_y}' x2='#{x_max}' y2='#{bus_y}' stroke='#{stroke}' />)
        other_branches = sorted[1..-1].map do |line|
          x = line.x1
          target_y = line.y2
          %(<line x1='#{x}' y1='#{bus_y}' x2='#{x}' y2='#{target_y}' stroke='#{stroke}' />\n) \
            + %(<polygon fill='#{stroke}' stroke='#{stroke}' points='#{x},#{target_y} #{x - 3},#{target_y - 6} #{x + 3},#{target_y - 6} #{x},#{target_y}' />)
        end.join("\n")
        branches_markup = "#{bus}\n#{other_branches}"
      end

      label_svg = %(<text fill='#{text_color}' text-anchor='middle' x='#{x_min}' y='#{y_top + 14}'>#{CGI.escapeHTML(name)}</text>)
      attributes, extras = CareeriaIdef0Boundary.decorate(primary, stroke)

      sorted[1..-1].each do |other_line|
        _other_attr, other_extra = CareeriaIdef0Boundary.decorate(other_line, stroke)
        if other_extra.include?('data-idef0-tunnel-marker')
          extras << other_extra.scan(/<g data-idef0-tunnel-marker='parent'.*?<\/g>/m).join
        end
      end

      markup = "#{trunk}\n#{branches_markup}\n#{label_svg}"
      "<g data-idef0-arrow='#{CGI.escapeHTML(name)}'#{attributes}>\n#{markup}\n#{extras}\n</g>"
    end
  end
end

module CareeriaIdef0Diagram
  def create_lines
    order_mode = CareeriaIdef0Style.order_mode
    if order_mode == 'heuristic'
      super
      return
    end

    source_boxes = @boxes.dup.sequence!
    source_lines = source_boxes.reduce(IDEF0::ArraySet.new) do |lines, target|
      source_boxes.each do |source|
        IDEF0::INTERNAL_LINE_TYPES.each do |line_type|
          line_type.make_line(source, target) { |line| lines.add(line) }
        end
      end
      lines
    end
    source_backward_count = source_lines.count(&:backward?)

    if order_mode == 'source' || order_mode == 'fixed' || source_backward_count.zero?
      @boxes = source_boxes
      @lines = source_lines
    else
      super
      heuristic_backward_count = @lines.count(&:backward?)
      if source_backward_count <= heuristic_backward_count
        @boxes = source_boxes
        @lines = source_lines
      end
    end

    @lines.each(&:attach)

    (IDEF0::EXTERNAL_LINE_TYPES + IDEF0::UNATTACHED_LINE_TYPES).each do |line_type|
      @boxes.each do |box|
        line_type.make_line(self, box) { |line| @lines.add(line.attach) }
      end
    end
  end

  def layout
    super
    if CareeriaIdef0Style.branching_mechanisms?
      mech_groups = @lines.select { |l| l.is_a?(IDEF0::ExternalMechanismLine) }.group_by(&:name)
      branched_count = mech_groups.count { |_, ls| ls.count > 1 }
      if branched_count > 1
        extra_clearance = (branched_count - 1) * 20
        @lines.select { |l| l.is_a?(IDEF0::ExternalMechanismLine) }.each do |line|
          line.instance_eval { add_clearance_from(@target.bottom_side, extra_clearance) }
        end
        resize(right_edge + 20, bottom_edge + 20 + extra_clearance)
      end
    end
  end

  def generate_boxes
    @boxes.map { |box| CareeriaIdef0Style.box_svg(box) }.join("\n")
  end

  def generate_lines
    result = []
    mech_lines = @lines.select { |l| l.is_a?(IDEF0::ExternalMechanismLine) }
    guidance_lines = @lines.select { |l| l.is_a?(IDEF0::ExternalGuidanceLine) }
    other_lines = @lines.reject { |l| l.is_a?(IDEF0::ExternalMechanismLine) || l.is_a?(IDEF0::ExternalGuidanceLine) }

    other_lines.each { |line| result << CareeriaIdef0Style.arrow_svg(line) }

    if CareeriaIdef0Style.branching_controls?
      guidance_groups = guidance_lines.group_by(&:name)
      bus_idx = 0
      guidance_groups.each do |name, lines|
        if lines.count > 1
          result << CareeriaIdef0Style.branched_guidance_svg(name, lines, bus_idx)
          bus_idx += 1
        else
          result << CareeriaIdef0Style.arrow_svg(lines.first)
        end
      end
    else
      guidance_lines.each { |l| result << CareeriaIdef0Style.arrow_svg(l) }
    end

    if CareeriaIdef0Style.branching_mechanisms?
      mech_groups = mech_lines.group_by(&:name)
      bus_idx = 0
      mech_groups.each do |name, lines|
        if lines.count > 1
          result << CareeriaIdef0Style.branched_mechanism_svg(name, lines, bus_idx)
          bus_idx += 1
        else
          result << CareeriaIdef0Style.arrow_svg(lines.first)
        end
      end
    else
      mech_lines.each { |l| result << CareeriaIdef0Style.arrow_svg(l) }
    end

    markup = result.join("\n")
    markup = apply_sharp_corners(markup) if CareeriaIdef0Style.corners_mode == 'sharp'
    apply_crossings(markup, CareeriaIdef0Style.crossings_mode)
  end

  def apply_sharp_corners(markup)
    markup.gsub(/<path\s+([^>]*\bd=[\x27"]([^\x27"]+)[\x27"][^>]*)(\/?>)/) do |full_match|
      attrs, d_str = $1, $2
      sharp_d = d_str.gsub(/([ML])\s+([-\d.]+)\s+([-\d.]+)\s+C\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)/) do
        cmd, xa, ya, cp1x, cp1y, _cp2x, _cp2y, xb, yb = $1, $2.to_f, $3.to_f, $4.to_f, $5.to_f, $6.to_f, $7.to_f, $8.to_f, $9.to_f
        if (ya - cp1y).abs < 0.5
          xc, yc = xb, ya
        else
          xc, yc = xa, yb
        end
        if cmd == 'M'
          "M #{xa.round(1)} #{ya.round(1)} L #{xc.round(1)} #{yc.round(1)} L #{xb.round(1)} #{yb.round(1)}"
        else
          "L #{xc.round(1)} #{yc.round(1)} L #{xb.round(1)} #{yb.round(1)}"
        end
      end
      sharp_d = sharp_d.gsub(/([ML])\s+([-\d.]+)\s+([-\d.]+)\s+Q\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)/) do
        cmd, xa, ya, cpx, cpy, xb, yb = $1, $2.to_f, $3.to_f, $4.to_f, $5.to_f, $6.to_f, $7.to_f
        if (ya - cpy).abs < 0.5
          xc, yc = xb, ya
        else
          xc, yc = xa, yb
        end
        if cmd == 'M'
          "M #{xa.round(1)} #{ya.round(1)} L #{xc.round(1)} #{yc.round(1)} L #{xb.round(1)} #{yb.round(1)}"
        else
          "L #{xc.round(1)} #{yc.round(1)} L #{xb.round(1)} #{yb.round(1)}"
        end
      end
      full_match.sub(/d=[\x27"][^\x27"]+[\x27"]/, %(d='#{sharp_d}'))
    end
  end

  def apply_crossings(svg_markup, mode, r = 5)
    return svg_markup if mode == 'none' || mode == false || mode == 'false'

    verticals = []
    svg_markup.scan(/<line\s+[^>]*x1=[\x27"]([^\x27"]+)[\x27"]\s+y1=[\x27"]([^\x27"]+)[\x27"]\s+x2=[\x27"]([^\x27"]+)[\x27"]\s+y2=[\x27"]([^\x27"]+)[\x27"][^>]*>/) do |x1, y1, x2, y2|
      fx1, fy1, fx2, fy2 = [x1, y1, x2, y2].map(&:to_f)
      if (fx1 - fx2).abs < 0.5 && (fy1 - fy2).abs > 2.0
        verticals << { x: fx1, y1: [fy1, fy2].min, y2: [fy1, fy2].max }
      end
    end

    svg_markup.scan(/<path\s+[^>]*d=[\x27"]([^\x27"]+)[\x27"][^>]*>/) do |d_match|
      d = d_match[0]
      tokens = d.scan(/([A-Za-z])\s*([-\d.\s,]*)/)
      cx, cy = 0.0, 0.0
      tokens.each do |cmd, coords|
        nums = coords.strip.split(/[\s,]+/).reject(&:empty?).map(&:to_f)
        case cmd
        when 'M'
          cx, cy = nums[0], nums[1] if nums.length >= 2
        when 'L'
          nums.each_slice(2) do |nx, ny|
            if (cx - nx).abs < 0.5 && (cy - ny).abs > 2.0
              verticals << { x: cx, y1: [cy, ny].min, y2: [cy, ny].max }
            end
            cx, cy = nx, ny
          end
        else
          cx, cy = nums[-2], nums[-1] if nums.length >= 2
        end
      end
    end

    transform_h_segment = lambda do |xs, xe, y|
      xmin, xmax = [xs, xe].min, [xs, xe].max
      direction = xs <= xe ? 1 : -1
      crosses = verticals.select do |v|
        v[:x] > (xmin + r + 2) && v[:x] < (xmax - r - 2) && v[:y1] < (y - 2) && v[:y2] > (y + 2)
      end.map { |v| v[:x] }.uniq.sort
      crosses.reverse! if direction == -1

      return nil if crosses.empty?

      parts = []
      crosses.each do |cx|
        if direction == 1
          parts << "L #{(cx - r).round(1)} #{y.round(1)}"
          if mode == 'arc'
            parts << "A #{r} #{r} 0 0 1 #{(cx + r).round(1)} #{y.round(1)}"
          else
            parts << "M #{(cx + r).round(1)} #{y.round(1)}"
          end
        else
          parts << "L #{(cx + r).round(1)} #{y.round(1)}"
          if mode == 'arc'
            parts << "A #{r} #{r} 0 0 1 #{(cx - r).round(1)} #{y.round(1)}"
          else
            parts << "M #{(cx - r).round(1)} #{y.round(1)}"
          end
        end
      end
      parts << "L #{xe.round(1)} #{y.round(1)}"
      parts
    end

    transformed = svg_markup.gsub(/<line\s+([^>]*x1=[\x27"]([^\x27"]+)[\x27"]\s+y1=[\x27"]([^\x27"]+)[\x27"]\s+x2=[\x27"]([^\x27"]+)[\x27"]\s+y2=[\x27"]([^\x27"]+)[\x27"][^>]*)(\/?>)/) do |full_match|
      all_attrs, x1_s, y1_s, x2_s, y2_s = $1, $2, $3, $4, $5
      x1, y1, x2, y2 = [x1_s, y1_s, x2_s, y2_s].map(&:to_f)
      if (y1 - y2).abs < 0.5 && (x1 - x2).abs > 2.0
        parts = transform_h_segment.call(x1, x2, y1)
        if parts
          stroke = all_attrs[/stroke=[\x27"][^\x27"]+[\x27"]/] || "stroke='black'"
          %(<path fill='none' #{stroke} d='M #{x1} #{y1} #{parts.join(" ")}' />)
        else
          full_match
        end
      else
        full_match
      end
    end

    transformed.gsub(/<path\s+([^>]*\bd=[\x27"]([^\x27"]+)[\x27"][^>]*)(\/?>)/) do |full_match|
      _attrs, d_str = $1, $2
      new_d = []
      tokens = d_str.scan(/([A-Za-z])\s*([-\d.\s,]*)/)
      cx, cy = 0.0, 0.0
      changed = false

      tokens.each do |cmd, coords|
        nums = coords.strip.split(/[\s,]+/).reject(&:empty?).map(&:to_f)
        case cmd
        when 'M'
          cx, cy = nums[0], nums[1] if nums.length >= 2
          new_d << "M #{cx} #{cy}"
        when 'L'
          nums.each_slice(2) do |nx, ny|
            if (cy - ny).abs < 0.5 && (cx - nx).abs > 2.0
              parts = transform_h_segment.call(cx, nx, cy)
              if parts
                changed = true
                new_d.concat(parts)
              else
                new_d << "L #{nx} #{ny}"
              end
            else
              new_d << "L #{nx} #{ny}"
            end
            cx, cy = nx, ny
          end
        when 'C'
          cx, cy = nums[-2], nums[-1] if nums.length >= 2
          new_d << "C #{nums.join(" ")}"
        when 'A'
          cx, cy = nums[-2], nums[-1] if nums.length >= 2
          new_d << "A #{nums.join(" ")}"
        when 'Q'
          cx, cy = nums[-2], nums[-1] if nums.length >= 2
          new_d << "Q #{nums.join(" ")}"
        else
          cx, cy = nums[-2], nums[-1] if nums.length >= 2
          new_d << "#{cmd} #{nums.join(" ")}"
        end
      end

      if changed
        full_match.sub(/d=[\x27"][^\x27"]+[\x27"]/, %(d='#{new_d.join(" ")}'))
      else
        full_match
      end
    end
  end
end

IDEF0::Diagram.prepend(CareeriaIdef0Diagram)

if __FILE__ == $PROGRAM_NAME
  options = { mode: 'decompose' }
  OptionParser.new do |parser|
    parser.on('--input PATH') { |value| options[:input] = value }
    parser.on('--style PATH') { |value| options[:style] = value }
    parser.on('--mode MODE') { |value| options[:mode] = value }
    parser.on('--process NAME') { |value| options[:process] = value }
    parser.on('--order ORDER') { |value| options[:order] = value }
    parser.on('--branching BRANCHING') { |value| options[:branching] = value }
    parser.on('--corners CORNERS') { |value| options[:corners] = value }
    parser.on('--crossings CROSSINGS') { |value| options[:crossings] = value }
  end.parse!

  abort 'missing --input' unless options[:input]
  abort 'mode must be auto, decompose, schematic or focus' unless %w[auto decompose schematic focus].include?(options[:mode])

  begin
    CareeriaIdef0Style.load(options[:style])
    CareeriaIdef0Style.settings['order'] = options[:order] if options[:order]
    CareeriaIdef0Style.settings['branching'] = options[:branching] if options[:branching]
    CareeriaIdef0Style.settings['corners'] = options[:corners] if options[:corners]
    CareeriaIdef0Style.settings['crossings'] = options[:crossings] if options[:crossings]
    statements = File.open(options[:input], 'r:utf-8') { |file| IDEF0::Statement.parse(file) }
    root = IDEF0::Process.parse(statements)
    CareeriaIdef0Boundary.validate(root)
    process = options[:process] ? root.find(options[:process]) : root
    abort "process not found: #{options[:process]}" unless process
    mode = options[:mode] == 'auto' ? (process.decomposable? ? 'decompose' : 'focus') : options[:mode]
    CareeriaIdef0Boundary.select_view(process, mode)
    puts process.public_send(mode).to_svg
  rescue StandardError => error
    warn "IDEF0-SVG: #{error.message}"
    exit 1
  end
end
