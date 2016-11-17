
#include "data.h"

Data::Data(std::string& s)
  : n(s)
{
}

const char*
Data::get() const
{
  return n.c_str();
}
//
