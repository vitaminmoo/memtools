# Ghidra script to recursively apply struct types to pointed-to memory
# @category MemTools
# @menupath Tools.Recursive Type Applier

from ghidra.program.model.data import PointerDataType, StructureDataType, ArrayDataType
from ghidra.program.model.symbol import SourceType

def get_pointer_target_type(field_dt):
    """Extract the target type from a pointer data type."""
    if not isinstance(field_dt, PointerDataType):
        return None
    return field_dt.getDataType()

def is_struct_pointer(field_dt):
    """Check if field is a pointer to a struct (not char* or void*)."""
    if not isinstance(field_dt, PointerDataType):
        return False
    target = field_dt.getDataType()
    if target is None:
        return False
    # char* and void* should not be recursively followed
    if target.getName() in ["char", "void", "undefined"]:
        return False
    # Unwrap arrays if needed
    while isinstance(target, ArrayDataType):
        target = target.getDataType()
    return isinstance(target, StructureDataType)

def read_pointer(address, ptr_size):
    """Read a pointer value from memory."""
    if ptr_size == 4:
        return toAddr(getInt(address) & 0xFFFFFFFF)
    else:
        return toAddr(getLong(address))

def apply_type_recursive(address, data_type, visited, ptr_size, depth=0):
    """
    Recursively apply data_type at address, then follow all struct pointers.

    Args:
        address: The address to apply the type at
        data_type: The DataType to apply
        visited: Set of already-visited addresses (for cycle detection)
        ptr_size: Pointer size (4 or 8)
        depth: Current recursion depth (for logging)

    Returns:
        Number of types applied
    """
    addr_long = address.getOffset()

    # Cycle detection
    if addr_long in visited:
        println("{}Skipping already visited: {}".format("  " * depth, address))
        return 0

    # Skip null pointers
    if addr_long == 0:
        return 0

    visited.add(addr_long)

    # Get the actual struct type (unwrap pointers/arrays)
    actual_type = data_type
    while isinstance(actual_type, PointerDataType):
        actual_type = actual_type.getDataType()
    while isinstance(actual_type, ArrayDataType):
        actual_type = actual_type.getDataType()

    if not isinstance(actual_type, StructureDataType):
        println("{}Not a struct type: {}".format("  " * depth, actual_type))
        return 0

    # Clear any existing data at this location
    clearListing(address, address.add(actual_type.getLength() - 1))

    # Apply the type
    try:
        createData(address, actual_type)
        println("{}Applied {} at {}".format("  " * depth, actual_type.getName(), address))
    except Exception as e:
        println("{}Failed to apply {} at {}: {}".format("  " * depth, actual_type.getName(), address, e))
        return 0

    count = 1

    # Now iterate through fields and follow struct pointers
    for i in range(actual_type.getNumComponents()):
        component = actual_type.getComponent(i)
        field_dt = component.getDataType()
        field_offset = component.getOffset()
        field_name = component.getFieldName()

        if is_struct_pointer(field_dt):
            target_type = get_pointer_target_type(field_dt)
            field_addr = address.add(field_offset)

            # Read the pointer value
            try:
                ptr_value = read_pointer(field_addr, ptr_size)
                if ptr_value.getOffset() != 0:
                    println("{}Following pointer {} -> {}".format("  " * depth, field_name or "unnamed", ptr_value))
                    count += apply_type_recursive(ptr_value, target_type, visited, ptr_size, depth + 1)
            except Exception as e:
                println("{}Failed to read pointer at {}: {}".format("  " * depth, field_addr, e))

    return count

def main():
    # Get pointer size from program
    ptr_size = currentProgram.getDefaultPointerSize()
    println("Pointer size: {} bytes".format(ptr_size))

    # Get the current location
    if currentLocation is None:
        popup("Please position cursor on a data location")
        return

    address = currentLocation.getAddress()
    println("Starting address: {}".format(address))

    # Get existing data at location
    data = getDataAt(address)
    if data is None:
        popup("No data defined at current location.\nPlease apply a struct type first.")
        return

    data_type = data.getDataType()
    println("Data type: {}".format(data_type.getName()))

    # Must be a struct
    actual_type = data_type
    while isinstance(actual_type, PointerDataType):
        actual_type = actual_type.getDataType()

    if not isinstance(actual_type, StructureDataType):
        popup("Current data must be a struct type, got: {}".format(data_type.getName()))
        return

    # Track visited addresses for cycle detection
    visited = set()

    # Start the recursive application
    # We don't re-apply at the starting address since it's already typed
    visited.add(address.getOffset())
    count = 0

    # Process children of the starting struct
    for i in range(actual_type.getNumComponents()):
        component = actual_type.getComponent(i)
        field_dt = component.getDataType()
        field_offset = component.getOffset()
        field_name = component.getFieldName()

        if is_struct_pointer(field_dt):
            target_type = get_pointer_target_type(field_dt)
            field_addr = address.add(field_offset)

            try:
                ptr_value = read_pointer(field_addr, ptr_size)
                if ptr_value.getOffset() != 0:
                    println("Following root pointer {} -> {}".format(field_name or "unnamed", ptr_value))
                    count += apply_type_recursive(ptr_value, target_type, visited, ptr_size, 1)
            except Exception as e:
                println("Failed to read pointer at {}: {}".format(field_addr, e))

    println("Done! Applied {} types.".format(count))
    popup("Recursively applied {} struct types".format(count))

main()
