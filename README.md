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
All you need is to `go get github.com/lukasjarosch/go-docx`

For some code, just go ahead and look into `./examples/simple/main.go`. 

### ➤ How it works
This section will give you a short overview of what's actually going on.
And honenstly.. it's a much needed reference for my future self :D.

#### Overview
The project does rely on some invariants of the WordprocessingML spec which defines the docx structure.
A good overview over the spec can be found on: [officeopenxml.com](http://officeopenxml.com/anatomyofOOXML.php).

Since this project aims to work only on text within the document, it currently only focuses on the **runs** (`<w:r>` element).
A run *always* encloses a **text** (`<w:t>` element) thus finding all runs inside the docx is the first step. Keep in
mind that a run does not need to have a text element. It can also contain an image for example. But all text
literals will always be inside a run, within their respective text tags.

To illustrate that, here is how this might look inside the document.xml.

```xml
 <w:p>
    <w:r>
        <w:t>{key-with-dashes}</w:t>
    </w:r>
</w:p>
```
One can clearly see that replacing the `{key-with-dashes}` placeholder is quite simple. 
Just do a `strings.Replace()`, right? **Wrong!**

Although this might work on 70-80% of the time, it will not work reliably.
The reason is how the WordprocessingML spec is set-up. It will fragment text-literals 
based on many different circumstances. 

For example if you added half of the placeholder, saved
and quit Word, and then add the second half of the placeholder, it might happen (in order to preserve session history), that the placeholder will look something like that (simplified).

```xml
 <w:p>
    <w:r>
        <w:t>{key-</w:t>
    </w:r>
    <w:r>
        <w:t>with-dashes}</w:t>
    </w:r>
</w:p>
```

As you can clearly see, doing a simple replace doesn't do it for this case.

