<h1 align="center">go-docx</h1>
<p align="center">
   <b>Replace placeholders inside docx documents with speed and confidence.</b>
   
   <p align="center"><sub>This project provides a simple and clean API to perform replacing of user-defined placeholders.   
   Without the uncertainty that the placeholders may be ripped  aparty by the WordprocessingML engine used to create the document.</sub></p>
</p>

<p align="center">
  <img src="https://github.com/lukasjarosch/go-docx/blob/refactoring/screenshot.png" alt="Example" width="900" />
</p>


* **Simple**: The API exposed is kept to a minimum in order to stick to the purpose.
* **Fast**: go-docx is fast since it operates directly on the byte contents instead mapping the XMLs to a custom data struct.
* **Zero dependencies**: go-docx is build with the go stdlib only.

### ➤ Purpose
The task at hand was to replace a set of user-defined placeholders inside a docx document archive with calculated values.
All current implementations in Golang which solve this problem use a naive approach by attempting to `strings.Replace()` the placeholders.

Due to the nature of the WordprocessingML specification, a placeholder which is defined as `{the-placeholder}` may be ripped apart inside the resulting XML.
The placeholder may then be in two fragments for example `{the-` and `placeholder}` which are spaced apart inside the XML.

The naive approach therefore is not always working. To provide a way to replace placeholders, even if they are fragmented, is the purpose of this library.

### ➤ Getting Started
