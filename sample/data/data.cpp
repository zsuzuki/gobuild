
#include "data.h"

Data::Data(std::string& s)
  : n(s)
{
#if TEST3
  n += "(TEST3)";
#endif
}

const char*
Data::get() const
{
  return n.c_str();
}
//
