(comment) @comment

[
  (addition)
  (new_file)
] @diff.plus

[
  (deletion)
  (old_file)
] @diff.minus

(commit) @constant

(location) @attribute

(command
  "diff" @function
  (argument) @variable.parameter)

(filename) @string.special.path

(mode) @constant.numeric

[
  ".."
  "+"
  "++"
  "+++"
  "++++"
  "-"
  "--"
  "---"
  "----"
] @punctuation.special

; Changed lines and file headers keep one color: the later pattern wins the same node
(addition _ @diff.plus)
(new_file _ @diff.plus)
(deletion _ @diff.minus)
(old_file _ @diff.minus)

[
  (binary_change)
  (similarity)
  (file_change)
] @label

(index
  "index" @keyword)

(similarity
  (score) @constant.numeric
  "%" @constant.numeric)
