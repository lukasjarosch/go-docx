<h1 align="center">go-docx</h1>
<p align="center">
<b>Zero dependency lib to replace delimited placeholders inside .docx files.</b><br/>
<sub>
   This library was created in order to have a stable way to replace placeholders inside docx files.
   Current solutions basically only attempt to ```strings.Replace()``` inside the document.xml.
   
   Due to the nature of WordprocessingML documents, this approach is not stable since
   placeholders can be ripped apart inside the xml.
   
   **go-docx** provides a way to replace all placeholders with confidence, even
   if they are separated within the documents.
</sub>
</p>

* **Simple**: The API exposed is kept to a minimum in order to stick to the purpose.
* **Fast**: go-docx is fast since it operates directly on the byte contents instead mapping the XMLs to a custom data struct.
* **Zero dependencies**: go-docx is build with the go stdlib only.
* **Reliable**: go-docx will reliably replace placeholders inside the document.