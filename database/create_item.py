"""
This module adds an item based on a prototype to a room.
"""

import uuid
from decimal import Decimal

import boto3
from botocore.exceptions import ClientError


def display_rooms(dynamodb) -> list:
    """
    Fetches and displays all rooms from the 'rooms' DynamoDB table.

    Args:
        dynamodb: The DynamoDB resource object.

    Returns:
        A list of room dictionaries.
    """
    try:
        table = dynamodb.Table("rooms")
        response = table.scan()
        rooms = response.get("Items", [])
        if not rooms:
            print("No rooms found.")
            return []

        print("Available Rooms:")
        for room in rooms:
            room_id = int(room["RoomID"])
            title = room.get("Title", "No Title")
            print(f"{room_id}: {title}")
        return rooms
    except ClientError as e:
        print(f"Error fetching rooms: {e.response['Error']['Message']}")
        return []


def prompt_for_room() -> int:
    """
    Prompts the user to enter a room ID.

    Returns:
        The room ID entered by the user, or None to quit.
    """
    while True:
        room_input: str = input("Enter room ID (X to quit): ").strip().upper()
        if room_input == "X":
            return None  # type: ignore
        try:
            return int(room_input)
        except ValueError:
            print("Please enter a valid number or 'X' to quit.")


def display_prototypes(dynamodb) -> list:
    """
    Fetches and displays all item prototypes from the 'prototypes' DynamoDB table.
    """
    try:
        table = dynamodb.Table("prototypes")
        response = table.scan()
        prototypes = response.get("Items", [])
        if not prototypes:
            print("No prototypes found.")
            return []

        print("Available Prototypes:")
        for prototype in prototypes:
            prototype_id = prototype.get("PrototypeID", "No ID")  # Changed from "id" to "PrototypeID"
            name = prototype.get("name", "No Name")
            print(f"{prototype_id}: {name}")
        return prototypes
    except ClientError as e:
        print(f"Error fetching prototypes: {e.response['Error']['Message']}")
        return []


def prompt_for_prototype() -> str:
    """
    Prompts the user to enter a prototype ID.

    Returns:
        The prototype ID entered by the user, or an empty string to cancel.
    """
    return input("Enter prototype ID (empty to cancel): ").strip()


def create_new_item_from_prototype(prototype: dict) -> dict:
    """
    Creates a new item based on the given prototype.

    Args:
        prototype: The prototype dictionary.

    Returns:
        A new item dictionary with a unique ID and properties copied from the prototype.
    """
    new_item: dict = {
        "ItemID": str(uuid.uuid4()),
        "Name": prototype.get("name", "Unnamed Item"),
        "Description": prototype.get("description", ""),
        "Mass": Decimal(str(prototype.get("mass", 0))),
        "Value": Decimal(str(prototype.get("value", 0))),
        "Stackable": prototype.get("stackable", False),
        "MaxStack": Decimal(str(prototype.get("max_stack", 1))),
        "Quantity": Decimal("1"),
        "Wearable": prototype.get("wearable", False),
        "WornOn": prototype.get("worn_on", []),
        "Verbs": prototype.get("verbs", {}),
        "Overrides": prototype.get("overrides", {}),
        "TraitMods": {k: Decimal(str(v)) for k, v in prototype.get("trait_mods", {}).items()},
        "Container": prototype.get("container", False),
        "IsPrototype": False,
        "IsWorn": False,
        "CanPickUp": prototype.get("can_pick_up", True),
        "Contents": [],
        "Metadata": prototype.get("metadata", {}),
    }
    return new_item


def ensure_item_ids_list(dynamodb, room_id: int) -> bool:
    """
    Ensures that the ItemIDs attribute exists in the room and is a list.
    If it doesn't exist, it creates an empty list.

    Args:
        dynamodb: The DynamoDB resource object.
        room_id: The ID of the room to check.

    Returns:
        True if the operation was successful, False otherwise.
    """
    rooms_table = dynamodb.Table("rooms")
    try:
        _ = rooms_table.update_item(
            Key={"RoomID": room_id},
            UpdateExpression="SET ItemIDs = if_not_exists(ItemIDs, :empty_list)",
            ExpressionAttributeValues={
                ":empty_list": [],
            },
            ReturnValues="UPDATED_NEW",
        )
        print(f"Ensured ItemIDs is a list for room {room_id}.")
        return True
    except ClientError as e:
        print(f"Error ensuring ItemIDs is a list: {e.response['Error']['Message']}")
        return False


def add_item_to_room(dynamodb, room: dict, new_item: dict) -> bool:
    """
    Adds the new item to the 'items' table and updates the room to include the item.

    Args:
        dynamodb: The DynamoDB resource object.
        room: The room dictionary where the item will be added.
        new_item: The item dictionary to add.

    Returns:
        True if the item was successfully added to the room, False otherwise.
    """
    items_table = dynamodb.Table("items")
    rooms_table = dynamodb.Table("rooms")

    # Step 1: Save new item to the 'items' table
    try:
        items_table.put_item(Item=new_item)
        print(f"Successfully added item '{new_item['Name']}' to items table.")
    except ClientError as e:
        print(f"Error saving new item to items table: {e.response['Error']['Message']}")
        return False

    # Step 2: Update the room to include the new item
    try:
        room_id = int(room["RoomID"])

        # First, get the current state of the room
        response = rooms_table.get_item(Key={"RoomID": room_id})
        current_room = response.get("Item", {})
        current_item_ids = current_room.get("ItemIDs", [])

        # Ensure current_item_ids is a list
        if current_item_ids is None:
            current_item_ids = []

        # Add the new item's ID to the room's ItemIDs list
        updated_item_ids = current_item_ids + [new_item["ItemID"]]

        # Update the room with the new ItemIDs list
        response = rooms_table.update_item(
            Key={"RoomID": room_id},
            UpdateExpression="SET ItemIDs = :item_ids",
            ExpressionAttributeValues={":item_ids": updated_item_ids},
            ReturnValues="UPDATED_NEW"
        )
        print(f"Successfully updated room {room_id}. New ItemIDs: {response['Attributes'].get('ItemIDs', [])}")
    except ClientError as e:
        print(f"Error updating room: {e.response['Error']['Message']}")
        # Attempt to roll back by deleting the item we just added
        try:
            items_table.delete_item(Key={"ItemID": new_item["ItemID"]})
            print(f"Rolled back: Deleted item '{new_item['Name']}' from items table.")
        except ClientError as del_e:
            print(f"Error rolling back item addition: {del_e.response['Error']['Message']}")
        return False

    print(f"Successfully added item '{new_item['Name']}' (ItemID: {new_item['ItemID']}) to room {room_id}")
    return True


def main() -> None:
    dynamodb = boto3.resource("dynamodb", region_name="us-east-1")  # Replace with your AWS region

    while True:
        rooms: list = display_rooms(dynamodb)
        if not rooms:
            print("No rooms available. Exiting.")
            break

        room_id: int = prompt_for_room()
        if room_id is None:
            print("Exiting.")
            break

        room = next((r for r in rooms if int(r["RoomID"]) == room_id), None)
        if not room:
            print("Room not found.")
            continue

        prototypes: list = display_prototypes(dynamodb)
        if not prototypes:
            print("No item prototypes found. Please add some prototypes first.")
            continue

        prototype_id: str = prompt_for_prototype()
        if not prototype_id:
            print("No prototype selected. Returning to room selection.")
            continue

        selected_prototype = next((p for p in prototypes if p.get("PrototypeID") == prototype_id), None)
        if not selected_prototype:
            print("Prototype not found.")
            continue

        new_item: dict = create_new_item_from_prototype(selected_prototype)
        if add_item_to_room(dynamodb, room, new_item):
            print(f"Successfully added '{new_item['Name']}' to room {room_id}.")
        else:
            print("Failed to add item to room.")


if __name__ == "__main__":
    main()
