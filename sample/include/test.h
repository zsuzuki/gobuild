#pragma once

#include <string>

class Test
{
  std::string str = "Hello,World";
public:
  const char* get() const { return str.c_str(); }
};

//
