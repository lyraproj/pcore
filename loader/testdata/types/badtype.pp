type BadType = Object[{
  attributes => {
    first => String,
    second => Integer # missing comma here
    third => Boolean
  }
}]