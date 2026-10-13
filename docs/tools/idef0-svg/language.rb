require 'cgi'

# Local language and text layout support around the vendored IDEF0-SVG engine.
module CareeriaIdef0Language
  VERBS = {
    'получает' => 'receives',
    'подчиняется' => 'respects',
    'создаёт' => 'produces',
    'создает' => 'produces',
    'производит' => 'produces',
    'использует' => 'requires',
    'требует' => 'requires',
    'состоит из' => 'is composed of'
  }.freeze
  VERB_PATTERN = /\s(#{VERBS.keys.sort_by { |word| -word.length }.map { |word| Regexp.escape(word) }.join('|')})\s/u

  def self.translate(text)
    text.each_line.map do |line|
      line.sub(VERB_PATTERN) { " #{VERBS.fetch(Regexp.last_match(1))} " }
    end.join
  end

  module StatementParser
    def parse(text)
      CareeriaIdef0Language.translate(text).each_line.with_index(1).flat_map do |line, number|
        begin
          statements = super(line)
          statements.each do |statement|
            unless %w[receives respects produces requires].include?(statement.predicate) || statement.predicate == 'is composed of'
              raise "unknown predicate #{statement.predicate.inspect}"
            end
          end
          statements
        rescue StandardError => error
          raise "line #{number}: #{error.message}"
        end
      end
    end
  end

  module Typography
    MAX_TEXT_WIDTH = 250
    LINE_HEIGHT = 18

    def self.measure(text)
      text.each_char.sum do |char|
        case char
        when ' ' then 4
        when /[ilI1.,:;'!|]/ then 4
        when /[MWЖШЩЮФЫ]/u then 11
        when /[А-Яа-яЁё]/u then 8
        when /[A-Z]/ then 8
        else 7
        end
      end
    end

    def self.lines(text)
      result = []
      current = ''
      text.split(/\s+/).each do |word|
        candidate = current.empty? ? word : "#{current} #{word}"
        if measure(candidate) <= MAX_TEXT_WIDTH
          current = candidate
          next
        end
        result << current unless current.empty?
        current = ''
        word.each_char do |char|
          if !current.empty? && measure(current + char) > MAX_TEXT_WIDTH
            result << current
            current = ''
          end
          current += char
        end
      end
      result << current unless current.empty?
      result.empty? ? [''] : result
    end
  end

  module LabelWidth
    def length(text)
      CareeriaIdef0Language::Typography.measure(text)
    end
  end

  module WrappedProcessBox
    def width
      text_width = CareeriaIdef0Language::Typography.lines(name)
        .map { |line| CareeriaIdef0Language::Typography.measure(line) }.max
      [text_width + 48, [top_side.anchor_count, bottom_side.anchor_count].max * 20 + 20].max
    end

    def height
      line_count = CareeriaIdef0Language::Typography.lines(name).length
      [60, line_count * CareeriaIdef0Language::Typography::LINE_HEIGHT + 28,
       [left_side.anchor_count, right_side.anchor_count].max * 20 + 20].max
    end

    def to_svg
      lines = CareeriaIdef0Language::Typography.lines(name)
      center_x = x1 + width / 2
      first_y = y1 + height / 2 - (lines.length - 1) * CareeriaIdef0Language::Typography::LINE_HEIGHT / 2 + 4
      spans = lines.each_with_index.map do |line, index|
        dy = index.zero? ? 0 : CareeriaIdef0Language::Typography::LINE_HEIGHT
        "<tspan x='#{center_x}' dy='#{dy}'>#{CGI.escapeHTML(line)}</tspan>"
      end.join
      "<rect x='#{x1}' y='#{y1}' width='#{width}' height='#{height}' fill='none' stroke='black' />\n" \
        "<text text-anchor='middle' x='#{center_x}' y='#{first_y}'>#{spans}</text>\n"
    end
  end
end

IDEF0::Statement.singleton_class.prepend(CareeriaIdef0Language::StatementParser)
IDEF0::Label.singleton_class.prepend(CareeriaIdef0Language::LabelWidth)
IDEF0::ProcessBox.prepend(CareeriaIdef0Language::WrappedProcessBox)
