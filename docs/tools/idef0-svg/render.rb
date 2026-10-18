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
      unknown = data.keys - %w[mode process views defaults blocks arrows boundaries order branching corners crossings radius icom callouts]
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

    attr_accessor :current_mech_layout, :current_guidance_layout

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

    def corner_radius
      return 0.0 if corners_mode == 'sharp'
      raw = settings&.fetch('radius', nil)
      raw ? raw.to_f : 8.0
    end

    def icom_enabled?
      raw = settings&.fetch('icom', false)
      return true if raw == true || raw == 'true' || raw == 'on' || raw == 'show' || raw == 'yes'
      false
    end

    def callouts_enabled?
      raw = settings&.fetch('callouts', true)
      return false if raw == false || raw == 'false' || raw == 'off' || raw == 'none' || raw == 'no'
      true
    end

    def compute_mechanism_labels_layout(lines)
      mech_lines = lines.to_a.select { |l| l.is_a?(IDEF0::ExternalMechanismLine) }
      return {} if mech_lines.empty?

      groups = mech_lines.group_by(&:name)
      branched_groups = groups.select { |_, ls| ls.count > 1 }
      return {} if branched_groups.empty? && !branching_mechanisms?

      placed = []
      groups.each do |name, ls|
        next if ls.count > 1 && branching_mechanisms?
        line = ls.first
        x = line.x1.to_f
        y = line.y1 - 5.0
        w = [name.length * 7.5, 36.0].max
        placed << { name: name, x_min: x - w / 2.0 - 6.0, x_max: x + w / 2.0 + 6.0, y_min: y - 14.0, y_max: y + 2.0 }
      end

      layout = {}
      branched_groups.keys.sort.each do |name|
        ls = groups[name]
        sorted_ls = ls.sort_by(&:x1)
        primary = sorted_ls.first
        x = primary.x1.to_f
        base_y = primary.y1 - 5.0
        w = [name.length * 7.5, 36.0].max
        x_min = x - w / 2.0 - 6.0
        x_max = x + w / 2.0 + 6.0

        row = 0
        curr_y = base_y
        loop do
          y_min = curr_y - 14.0
          y_max = curr_y + 2.0
          collision = placed.any? do |p|
            x_min < p[:x_max] && x_max > p[:x_min] && y_min < p[:y_max] && y_max > p[:y_min]
          end
          if collision
            row += 1
            curr_y = base_y + row * 18.0
          else
            placed << { name: name, x_min: x_min, x_max: x_max, y_min: y_min, y_max: y_max }
            layout[name] = { name: name, x: x, y: curr_y, row: row }
            break
          end
        end
      end
      layout
    end

    def compute_guidance_labels_layout(lines)
      guidance_lines = lines.to_a.select { |l| l.is_a?(IDEF0::ExternalGuidanceLine) }
      return {} if guidance_lines.empty?

      groups = guidance_lines.group_by(&:name)
      branched_groups = groups.select { |_, ls| ls.count > 1 }
      return {} if branched_groups.empty? && !branching_controls?

      placed = []
      groups.each do |name, ls|
        next if ls.count > 1 && branching_controls?
        line = ls.first
        x = line.x1.to_f
        y = line.y1 + 15.0
        w = [name.length * 7.5, 36.0].max
        placed << { name: name, x_min: x - w / 2.0 - 6.0, x_max: x + w / 2.0 + 6.0, y_min: y - 14.0, y_max: y + 2.0 }
      end

      layout = {}
      branched_groups.keys.sort.each do |name|
        ls = groups[name]
        sorted_ls = ls.sort_by(&:x1)
        primary = sorted_ls.first
        x = primary.x1.to_f
        base_y = primary.y1 + 15.0
        w = [name.length * 7.5, 36.0].max
        x_min = x - w / 2.0 - 6.0
        x_max = x + w / 2.0 + 6.0

        row = 0
        curr_y = base_y
        loop do
          y_min = curr_y - 14.0
          y_max = curr_y + 2.0
          collision = placed.any? do |p|
            x_min < p[:x_max] && x_max > p[:x_min] && y_min < p[:y_max] && y_max > p[:y_min]
          end
          if collision
            row += 1
            curr_y = base_y - row * 18.0
          else
            placed << { name: name, x_min: x_min, x_max: x_max, y_min: y_min, y_max: y_max }
            layout[name] = { name: name, x: x, y: curr_y, row: row }
            break
          end
        end
      end
      layout
    end

    def compute_mechanism_bus_indices(lines)
      return {} unless branching_mechanisms?
      mech_lines = lines.to_a.select { |l| l.is_a?(IDEF0::ExternalMechanismLine) }
      groups = mech_lines.group_by(&:name).select { |_, ls| ls.count > 1 }
      return {} if groups.empty?

      sorted_names = groups.keys.sort_by do |name|
        ls = groups[name]
        seqs = ls.map { |l| l.target ? l.target.sequence : 0 }
        min_seq = seqs.min || 0
        max_seq = seqs.max || 0
        [-min_seq, -(max_seq - min_seq), name]
      end

      indices = {}
      sorted_names.each_with_index { |name, idx| indices[name] = idx }
      indices
    end

    def compute_guidance_bus_indices(lines)
      return {} unless branching_controls?
      guidance_lines = lines.to_a.select { |l| l.is_a?(IDEF0::ExternalGuidanceLine) }
      groups = guidance_lines.group_by(&:name).select { |_, ls| ls.count > 1 }
      return {} if groups.empty?

      sorted_names = groups.keys.sort_by do |name|
        ls = groups[name]
        seqs = ls.map { |l| l.target ? l.target.sequence : 0 }
        min_seq = seqs.min || 0
        max_seq = seqs.max || 0
        [-min_seq, -(max_seq - min_seq), name]
      end

      indices = {}
      sorted_names.each_with_index { |name, idx| indices[name] = idx }
      indices
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
        .gsub('<text ', "<text paint-order='stroke fill' stroke='#FFFFFF' stroke-width='4' stroke-linejoin='round' fill='#{text}' ")

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
      y_bottom = primary.y1
      boxes_bottom = sorted.map(&:y2).max

      bus_y = boxes_bottom + 15 + bus_index * 20
      rounded = corners_mode == 'rounded'
      r = CareeriaIdef0Style.corner_radius
      cp = (r * 0.5).round(1)

      layout_info = CareeriaIdef0Style.current_mech_layout&.fetch(name, nil) || { row: 0 }
      row = layout_info[:row] || 0
      base_label_y = y_bottom - 5
      label_y = layout_info[:y] || (base_label_y + row * 18)

      first_target_y = sorted.first.y2
      trunk_start_y = (row == 1) ? (label_y - 15) : (y_bottom - 20)
      trunk = %(<line x1='#{x_min}' y1='#{trunk_start_y}' x2='#{x_min}' y2='#{first_target_y}' stroke='#{stroke}' />\n) \
        + %(<polygon fill='#{stroke}' stroke='#{stroke}' points='#{x_min},#{first_target_y} #{x_min - 3},#{first_target_y + 6} #{x_min + 3},#{first_target_y + 6} #{x_min},#{first_target_y}' />)

      if rounded && r > 0 && (x_max - x_min) >= r * 2
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

      callout_svg = ''
      if row >= 2 && CareeriaIdef0Style.callouts_enabled?
        callout_svg = %(<line x1='#{x_min}' y1='#{y_bottom - 20}' x2='#{x_min}' y2='#{label_y - 12}' stroke='#{stroke}' stroke-dasharray='3,3' stroke-width='1' />\n)
      end

      label_svg = %(#{callout_svg}<text paint-order='stroke fill' stroke='#FFFFFF' stroke-width='4' stroke-linejoin='round' fill='#{text_color}' text-anchor='middle' x='#{x_min}' y='#{label_y}'>#{CGI.escapeHTML(name)}</text>)
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
      y_top = primary.y1
      boxes_top = sorted.map(&:y2).min

      bus_y = boxes_top - 15 - bus_index * 20
      rounded = corners_mode == 'rounded'
      r = CareeriaIdef0Style.corner_radius
      cp = (r * 0.5).round(1)

      layout_info = CareeriaIdef0Style.current_guidance_layout&.fetch(name, nil) || { row: 0 }
      row = layout_info[:row] || 0
      base_label_y = y_top + 15
      label_y = layout_info[:y] || (base_label_y - row * 18)

      first_target_y = sorted.first.y2
      trunk_start_y = (row == 1) ? (label_y + 15) : (y_top + 20)
      trunk = %(<line x1='#{x_min}' y1='#{trunk_start_y}' x2='#{x_min}' y2='#{first_target_y}' stroke='#{stroke}' />\n) \
        + %(<polygon fill='#{stroke}' stroke='#{stroke}' points='#{x_min},#{first_target_y} #{x_min - 3},#{first_target_y - 6} #{x_min + 3},#{first_target_y - 6} #{x_min},#{first_target_y}' />)

      if rounded && r > 0 && (x_max - x_min) >= r * 2
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

      callout_svg = ''
      if row >= 2 && CareeriaIdef0Style.callouts_enabled?
        callout_svg = %(<line x1='#{x_min}' y1='#{y_top + 20}' x2='#{x_min}' y2='#{label_y + 4}' stroke='#{stroke}' stroke-dasharray='3,3' stroke-width='1' />\n)
      end

      label_svg = %(#{callout_svg}<text paint-order='stroke fill' stroke='#FFFFFF' stroke-width='4' stroke-linejoin='round' fill='#{text_color}' text-anchor='middle' x='#{x_min}' y='#{label_y}'>#{CGI.escapeHTML(name)}</text>)
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

    mech_layout = CareeriaIdef0Style.compute_mechanism_labels_layout(@lines)
    max_mech_row = mech_layout.values.map { |l| l[:row] }.max || 0
    if max_mech_row > 0
      extra_h = max_mech_row * 18 + 10
      resize(right_edge + 20, bottom_edge + 20 + extra_h)
    end
  end

  def sequence_anchors
    mech_bus_indices = CareeriaIdef0Style.compute_mechanism_bus_indices(@lines)
    guidance_bus_indices = CareeriaIdef0Style.compute_guidance_bus_indices(@lines)

    @boxes.each do |box|
      sequence_side_anchors(box.bottom_side, :bottom, mech_bus_indices, guidance_bus_indices)
      sequence_side_anchors(box.top_side, :top, mech_bus_indices, guidance_bus_indices)
      sequence_side_anchors(box.left_side, :left, mech_bus_indices, guidance_bus_indices)
      sequence_side_anchors(box.right_side, :right, mech_bus_indices, guidance_bus_indices)
    end
  end

  def sequence_side_anchors(side, side_type, mech_bus_indices, guidance_bus_indices)
    anchors = side.instance_variable_get(:@anchors)
    return side.sequence_anchors if anchors.nil?

    sorted = anchors.to_a.sort_by do |anchor|
      anchor_precedence_for(anchor, side, side_type, mech_bus_indices, guidance_bus_indices)
    end
    side.instance_variable_set(:@anchors, IDEF0::ArraySet.new(sorted).sequence!)
  end

  def anchor_precedence_for(anchor, side, side_type, mech_bus_indices, guidance_bus_indices)
    lines = anchor.lines
    return [99, 99, anchor.name] if lines.nil? || lines.empty?

    lines.map do |line|
      line_precedence_for(line, anchor, side, side_type, mech_bus_indices, guidance_bus_indices)
    end.min
  end

  def line_precedence_for(line, anchor, side, side_type, mech_bus_indices, guidance_bus_indices)
    box = side.instance_variable_get(:@box)
    case side_type
    when :bottom
      if line.is_a?(IDEF0::ExternalMechanismLine)
        if mech_bus_indices.key?(line.name)
          same = @lines.select { |l| l.is_a?(IDEF0::ExternalMechanismLine) && l.name == line.name }
          min_seq = same.map { |l| l.target ? l.target.sequence : 0 }.min || 0
          curr_seq = box ? box.sequence : 0
          if curr_seq == min_seq
            [2, mech_bus_indices[line.name], anchor.name]
          else
            [0, mech_bus_indices[line.name], anchor.name]
          end
        else
          [1, 0, anchor.name]
        end
      elsif line.is_a?(IDEF0::ForwardMechanismLine)
        src_seq = line.source ? line.source.sequence : 0
        [0, 100 - src_seq, anchor.name]
      elsif line.is_a?(IDEF0::BackwardMechanismLine)
        src_seq = line.source ? line.source.sequence : 0
        [3, src_seq, anchor.name]
      else
        [1, 1, anchor.name]
      end

    when :top
      if line.is_a?(IDEF0::ExternalGuidanceLine)
        if guidance_bus_indices.key?(line.name)
          same = @lines.select { |l| l.is_a?(IDEF0::ExternalGuidanceLine) && l.name == line.name }
          min_seq = same.map { |l| l.target ? l.target.sequence : 0 }.min || 0
          curr_seq = box ? box.sequence : 0
          if curr_seq == min_seq
            [2, guidance_bus_indices[line.name], anchor.name]
          else
            [0, guidance_bus_indices[line.name], anchor.name]
          end
        else
          [1, 0, anchor.name]
        end
      elsif line.is_a?(IDEF0::ForwardGuidanceLine)
        src_seq = line.source ? line.source.sequence : 0
        [0, 100 - src_seq, anchor.name]
      elsif line.is_a?(IDEF0::BackwardGuidanceLine)
        src_seq = line.source ? line.source.sequence : 0
        [3, src_seq, anchor.name]
      else
        [1, 1, anchor.name]
      end

    when :left
      if line.is_a?(IDEF0::ForwardInputLine)
        src_seq = line.source ? line.source.sequence : 0
        [0, 100 - src_seq, anchor.name]
      elsif line.is_a?(IDEF0::ExternalInputLine)
        [1, 0, anchor.name]
      elsif line.is_a?(IDEF0::BackwardInputLine)
        src_seq = line.source ? line.source.sequence : 0
        [2, src_seq, anchor.name]
      else
        [1, 1, anchor.name]
      end

    when :right
      if line.is_a?(IDEF0::BackwardGuidanceLine)
        tgt_seq = line.target ? line.target.sequence : 0
        [0, tgt_seq, anchor.name]
      elsif line.is_a?(IDEF0::ExternalOutputLine)
        [1, 0, anchor.name]
      elsif line.is_a?(IDEF0::ForwardInputLine) || line.is_a?(IDEF0::ForwardGuidanceLine) ||
            line.is_a?(IDEF0::ForwardMechanismLine) || line.is_a?(IDEF0::BackwardInputLine) ||
            line.is_a?(IDEF0::BackwardMechanismLine)
        tgt_seq = line.target ? line.target.sequence : 0
        [2, -tgt_seq, anchor.name]
      else
        [1, 1, anchor.name]
      end
    end
  end

  def generate_boxes
    @boxes.map { |box| CareeriaIdef0Style.box_svg(box) }.join("\n")
  end

  def generate_lines
    CareeriaIdef0Style.current_mech_layout = CareeriaIdef0Style.compute_mechanism_labels_layout(@lines)
    CareeriaIdef0Style.current_guidance_layout = CareeriaIdef0Style.compute_guidance_labels_layout(@lines)

    result = []
    mech_lines = @lines.select { |l| l.is_a?(IDEF0::ExternalMechanismLine) }
    guidance_lines = @lines.select { |l| l.is_a?(IDEF0::ExternalGuidanceLine) }
    other_lines = @lines.reject { |l| l.is_a?(IDEF0::ExternalMechanismLine) || l.is_a?(IDEF0::ExternalGuidanceLine) }

    other_lines.each { |line| result << CareeriaIdef0Style.arrow_svg(line) }

    if CareeriaIdef0Style.branching_controls?
      guidance_groups = guidance_lines.group_by(&:name)
      guidance_bus_indices = CareeriaIdef0Style.compute_guidance_bus_indices(@lines)
      guidance_groups.keys.sort.each do |name|
        lines = guidance_groups[name]
        if lines.count > 1
          bus_idx = guidance_bus_indices[name] || 0
          result << CareeriaIdef0Style.branched_guidance_svg(name, lines, bus_idx)
        else
          result << CareeriaIdef0Style.arrow_svg(lines.first)
        end
      end
    else
      guidance_lines.each { |l| result << CareeriaIdef0Style.arrow_svg(l) }
    end

    if CareeriaIdef0Style.branching_mechanisms?
      mech_groups = mech_lines.group_by(&:name)
      mech_bus_indices = CareeriaIdef0Style.compute_mechanism_bus_indices(@lines)
      mech_groups.keys.sort.each do |name|
        lines = mech_groups[name]
        if lines.count > 1
          bus_idx = mech_bus_indices[name] || 0
          result << CareeriaIdef0Style.branched_mechanism_svg(name, lines, bus_idx)
        else
          result << CareeriaIdef0Style.arrow_svg(lines.first)
        end
      end
    else
      mech_lines.each { |l| result << CareeriaIdef0Style.arrow_svg(l) }
    end

    markup = result.join("\n")
    if CareeriaIdef0Style.corners_mode == 'sharp'
      markup = apply_sharp_corners(markup)
    elsif (CareeriaIdef0Style.corner_radius - 10.0).abs > 0.05
      markup = apply_corner_radius(markup, CareeriaIdef0Style.corner_radius)
    end
    apply_crossings(markup, CareeriaIdef0Style.crossings_mode)
  end

  def apply_corner_radius(markup, r)
    return markup if (r - 10.0).abs < 0.05 || r <= 0.0

    cp = (r * 0.5).round(1)
    markup.gsub(/L\s+([-\d.]+)\s+([-\d.]+)\s+C\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)\s+([-\d.]+)/) do
      xa, ya, cp1x, cp1y, _cp2x, _cp2y, xb, yb = [$1, $2, $3, $4, $5, $6, $7, $8].map(&:to_f)
      if (ya - cp1y).abs < 0.5
        xc, yc = xb, ya
        dir_x = (xa < xc) ? -1 : 1
        dir_y = (yb > yc) ? 1 : -1
        new_xa = xc + dir_x * r
        new_ya = yc
        new_cp1x = xc + dir_x * cp
        new_cp1y = yc
        new_cp2x = xc
        new_cp2y = yc + dir_y * cp
        new_xb = xc
        new_yb = yc + dir_y * r
        %(L #{new_xa.round(1)} #{new_ya.round(1)} C #{new_cp1x.round(1)} #{new_cp1y.round(1)} #{new_cp2x.round(1)} #{new_cp2y.round(1)} #{new_xb.round(1)} #{new_yb.round(1)})
      else
        xc, yc = xa, yb
        dir_y = (ya < yc) ? -1 : 1
        dir_x = (xb > xc) ? 1 : -1
        new_xa = xc
        new_ya = yc + dir_y * r
        new_cp1x = xc
        new_cp1y = yc + dir_y * cp
        new_cp2x = xc + dir_x * cp
        new_cp2y = yc
        new_xb = xc + dir_x * r
        new_yb = yc
        %(L #{new_xa.round(1)} #{new_ya.round(1)} C #{new_cp1x.round(1)} #{new_cp1y.round(1)} #{new_cp2x.round(1)} #{new_cp2y.round(1)} #{new_xb.round(1)} #{new_yb.round(1)})
      end
    end
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
            parts << "A #{r} #{r} 0 0 0 #{(cx - r).round(1)} #{y.round(1)}"
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

IDEF0::Anchor.class_eval do
  attr_reader :lines, :side
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
    parser.on('--radius RADIUS') { |value| options[:radius] = value }
    parser.on('--crossings CROSSINGS') { |value| options[:crossings] = value }
    parser.on('--icom ICOM') { |value| options[:icom] = value }
    parser.on('--callouts CALLOUTS') { |value| options[:callouts] = value }
  end.parse!

  abort 'missing --input' unless options[:input]
  abort 'mode must be auto, decompose, schematic or focus' unless %w[auto decompose schematic focus].include?(options[:mode])

  begin
    CareeriaIdef0Style.load(options[:style])
    CareeriaIdef0Style.settings['order'] = options[:order] if options[:order]
    CareeriaIdef0Style.settings['branching'] = options[:branching] if options[:branching]
    CareeriaIdef0Style.settings['corners'] = options[:corners] if options[:corners]
    CareeriaIdef0Style.settings['radius'] = options[:radius] if options[:radius]
    CareeriaIdef0Style.settings['crossings'] = options[:crossings] if options[:crossings]
    CareeriaIdef0Style.settings['icom'] = options[:icom] if options[:icom]
    CareeriaIdef0Style.settings['callouts'] = options[:callouts] if options[:callouts]
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
