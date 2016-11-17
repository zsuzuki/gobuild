//
#pragma once

#include <string>
#include <vector>

class Data
{
  std::string n;
public:
  Data(std::string& s);
  const char* get() const;
};

using DataList = std::vector<Data>;
//
